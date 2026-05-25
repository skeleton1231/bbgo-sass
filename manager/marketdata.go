package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	pb "github.com/c9s/bbgo/saas/manager/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type pooledConn struct {
	conn *grpc.ClientConn
	ref  int
}

// MarketDataHub manages gRPC connections to bbgo containers and broadcasts
// data to connected WebSocket clients.
type MarketDataHub struct {
	mu       sync.RWMutex
	conn     *grpc.ClientConn
	market   pb.MarketDataServiceClient
	addr     string
	clients  map[string]map[chan json.RawMessage]struct{} // key: "market" or "user:{userID}"
	userPool map[string]*pooledConn                       // key: container address, shared gRPC connections
	dialFn   func(addr string) (*grpc.ClientConn, error)
	done     chan struct{}
}

func NewMarketDataHub(addr string, subs []MarketSub) (*MarketDataHub, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial marketdata %s: %w", addr, err)
	}

	hub := &MarketDataHub{
		conn:     conn,
		market:   pb.NewMarketDataServiceClient(conn),
		addr:     addr,
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		dialFn:   defaultDial,
		done:     make(chan struct{}),
	}

	go hub.subscribeDefault(subs)
	return hub, nil
}

func defaultDial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (h *MarketDataHub) getOrDial(addr string) (*grpc.ClientConn, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if pc, ok := h.userPool[addr]; ok {
		pc.ref++
		return pc.conn, nil
	}

	conn, err := h.dialFn(addr)
	if err != nil {
		return nil, err
	}
	h.userPool[addr] = &pooledConn{conn: conn, ref: 1}
	return conn, nil
}

func (h *MarketDataHub) releaseConn(addr string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	pc, ok := h.userPool[addr]
	if !ok {
		return
	}
	pc.ref--
	if pc.ref <= 0 {
		pc.conn.Close()
		delete(h.userPool, addr)
	}
}

func channelPb(name string) pb.Channel {
	switch name {
	case "trade":
		return pb.Channel_TRADE
	case "kline":
		return pb.Channel_KLINE
	case "book":
		return pb.Channel_BOOK
	default:
		return pb.Channel_TRADE
	}
}

func (h *MarketDataHub) subscribeDefault(subs []MarketSub) {
	const (
		minBackoff = 2 * time.Second
		maxBackoff = 30 * time.Second
	)

	pbSubs := make([]*pb.Subscription, len(subs))
	for i, s := range subs {
		pbSubs[i] = &pb.Subscription{
			Exchange: s.Exchange,
			Channel:  channelPb(s.Channel),
			Symbol:   s.Symbol,
			Interval: s.Interval,
			Depth:    s.Depth,
		}
	}

	backoff := minBackoff
	req := &pb.SubscribeRequest{Subscriptions: pbSubs}

	for {
		h.redial()
		select {
		case <-h.done:
			return
		case <-time.After(backoff):
		}

		ctx := context.Background()
		h.mu.RLock()
		client := h.market
		h.mu.RUnlock()
		stream, err := client.Subscribe(ctx, req)
		if err != nil {
			log.Printf("marketdata subscribe failed: %v, reconnecting in %v", err, backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		backoff = minBackoff
		for {
			md, err := stream.Recv()
			if err != nil {
				log.Printf("marketdata stream error: %v, reconnecting in %v", err, backoff)
				break
			}

			msg, err := json.Marshal(marketDataToJSON(md))
			if err != nil {
				log.Printf("marketdata json marshal error: %v", err)
				continue
			}
			h.broadcast("market", msg)
		}

		backoff = min(backoff*2, maxBackoff)
	}
}

func (h *MarketDataHub) redial() {
	h.mu.Lock()
	oldConn := h.conn
	if oldConn == nil {
		h.mu.Unlock()
		return
	}

	conn, err := h.dialFn(h.addr)
	if err != nil {
		h.mu.Unlock()
		log.Printf("marketdata redial failed: %v", err)
		return
	}
	h.conn = conn
	h.market = pb.NewMarketDataServiceClient(conn)
	h.mu.Unlock()

	oldConn.Close()
}

func (h *MarketDataHub) broadcast(key string, msg json.RawMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[key] {
		select {
		case ch <- msg:
		default:
		}
	}
}

// SubscribeUserData connects to a per-user bbgo container's gRPC server
// and streams order/trade/balance updates to the subscriber.
// gRPC connections are pooled: one connection per container address, shared across WS clients.
func (h *MarketDataHub) SubscribeUserData(ctx context.Context, userID string, containerAddr string, sessions []string) (chan json.RawMessage, error) {
	key := "user:" + userID

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions configured for user %s", userID)
	}

	conn, err := h.getOrDial(containerAddr)
	if err != nil {
		return nil, fmt.Errorf("dial user container %s: %w", containerAddr, err)
	}

	ch := make(chan json.RawMessage, 64)
	h.mu.Lock()
	if h.clients[key] == nil {
		h.clients[key] = make(map[chan json.RawMessage]struct{})
	}
	h.clients[key][ch] = struct{}{}
	h.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(sessions))

	for _, session := range sessions {
		go func(sessionName string) {
			defer wg.Done()
			h.receiveUserData(ctx, conn, userID, sessionName, ch)
		}(session)
	}

	go func() {
		wg.Wait()
		h.releaseConn(containerAddr)
		h.mu.Lock()
		delete(h.clients[key], ch)
		if len(h.clients[key]) == 0 {
			delete(h.clients, key)
		}
		h.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

func (h *MarketDataHub) receiveUserData(ctx context.Context, conn *grpc.ClientConn, userID string, session string, ch chan<- json.RawMessage) {
	client := pb.NewUserDataServiceClient(conn)
	stream, err := client.Subscribe(ctx, &pb.UserDataRequest{Session: session})
	if err != nil {
		log.Printf("user data subscribe %s/%s error: %v", userID, session, err)
		return
	}

	for {
		ud, err := stream.Recv()
		if err != nil {
			log.Printf("user data stream %s/%s error: %v", userID, session, err)
			return
		}
		msg, err := json.Marshal(userDataToJSON(ud))
		if err != nil {
			log.Printf("user data json marshal error for %s/%s: %v", userID, session, err)
			continue
		}
		select {
		case ch <- msg:
		default:
			log.Printf("user data channel full, dropping message for %s/%s", userID, session)
		}
	}
}

func (h *MarketDataHub) SubscribeMarket(ctx context.Context) (chan json.RawMessage, error) {
	key := "market"
	ch := make(chan json.RawMessage, 64)
	h.mu.Lock()
	if h.clients[key] == nil {
		h.clients[key] = make(map[chan json.RawMessage]struct{})
	}
	h.clients[key][ch] = struct{}{}
	h.mu.Unlock()

	return ch, nil
}

func (h *MarketDataHub) Unsubscribe(key string, ch chan json.RawMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.clients[key]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.clients, key)
		}
	}
}

func (h *MarketDataHub) Close() {
	if h.done != nil {
		select {
		case <-h.done:
		default:
			close(h.done)
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn != nil {
		h.conn.Close()
	}
	for addr, pc := range h.userPool {
		pc.conn.Close()
		delete(h.userPool, addr)
	}
}

func marketDataToJSON(md *pb.MarketData) map[string]interface{} {
	result := map[string]interface{}{
		"exchange": md.Exchange,
		"symbol":   md.Symbol,
		"channel":  md.Channel.String(),
		"event":    md.Event.String(),
	}

	if md.Depth != nil {
		result["depth"] = map[string]interface{}{
			"bids": priceVolumeSlice(md.Depth.Bids),
			"asks": priceVolumeSlice(md.Depth.Asks),
		}
	}
	if md.Kline != nil {
		result["kline"] = map[string]interface{}{
			"open":      md.Kline.Open,
			"high":      md.Kline.High,
			"low":       md.Kline.Low,
			"close":     md.Kline.Close,
			"volume":    md.Kline.Volume,
			"startTime": md.Kline.StartTime,
			"closed":    md.Kline.Closed,
		}
	}
	if md.Ticker != nil {
		result["ticker"] = map[string]interface{}{
			"open":   md.Ticker.Open,
			"high":   md.Ticker.High,
			"low":    md.Ticker.Low,
			"close":  md.Ticker.Close,
			"volume": md.Ticker.Volume,
		}
	}
	if len(md.Trades) > 0 {
		trades := make([]map[string]interface{}, len(md.Trades))
		for i, t := range md.Trades {
			trades[i] = map[string]interface{}{
				"id":        t.Id,
				"price":     t.Price,
				"quantity":  t.Quantity,
				"createdAt": t.CreatedAt,
				"side":      t.Side.String(),
			}
		}
		result["trades"] = trades
	}
	return result
}

func userDataToJSON(ud *pb.UserData) map[string]interface{} {
	result := map[string]interface{}{
		"session":  ud.Session,
		"exchange": ud.Exchange,
		"channel":  ud.Channel.String(),
		"event":    ud.Event.String(),
	}
	if len(ud.Balances) > 0 {
		balances := make([]map[string]string, len(ud.Balances))
		for i, b := range ud.Balances {
			balances[i] = map[string]string{
				"currency":  b.Currency,
				"available": b.Available,
				"locked":    b.Locked,
			}
		}
		result["balances"] = balances
	}
	if len(ud.Orders) > 0 {
		orders := make([]map[string]interface{}, len(ud.Orders))
		for i, o := range ud.Orders {
			orders[i] = map[string]interface{}{
				"id":               o.Id,
				"symbol":           o.Symbol,
				"side":             o.Side.String(),
				"price":            o.Price,
				"quantity":         o.Quantity,
				"executedQuantity": o.ExecutedQuantity,
				"status":           o.Status,
			}
		}
		result["orders"] = orders
	}
	if len(ud.Trades) > 0 {
		trades := make([]map[string]interface{}, len(ud.Trades))
		for i, t := range ud.Trades {
			trades[i] = map[string]interface{}{
				"id":       t.Id,
				"price":    t.Price,
				"quantity": t.Quantity,
				"side":     t.Side.String(),
				"fee":      t.Fee,
			}
		}
		result["trades"] = trades
	}
	return result
}

func priceVolumeSlice(pv []*pb.PriceVolume) []map[string]string {
	result := make([]map[string]string, len(pv))
	for i, p := range pv {
		result[i] = map[string]string{"price": p.Price, "volume": p.Volume}
	}
	return result
}
