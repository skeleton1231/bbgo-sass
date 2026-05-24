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

// MarketDataHub manages gRPC connections to bbgo containers and broadcasts
// data to connected WebSocket clients.
type MarketDataHub struct {
	mu      sync.RWMutex
	market  pb.MarketDataServiceClient
	conn    *grpc.ClientConn
	clients map[string]map[chan json.RawMessage]struct{} // key: "market" or "user:{userID}"
}

func NewMarketDataHub(addr string) (*MarketDataHub, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial marketdata %s: %w", addr, err)
	}

	hub := &MarketDataHub{
		market:  pb.NewMarketDataServiceClient(conn),
		conn:    conn,
		clients: make(map[string]map[chan json.RawMessage]struct{}),
	}

	go hub.subscribeDefault()
	return hub, nil
}

// subscribeDefault subscribes to common market data with automatic reconnect.
func (h *MarketDataHub) subscribeDefault() {
	const (
		minBackoff = 2 * time.Second
		maxBackoff = 30 * time.Second
	)

	backoff := minBackoff
	req := &pb.SubscribeRequest{
		Subscriptions: []*pb.Subscription{
			{Exchange: "binance", Channel: pb.Channel_TRADE, Symbol: "BTCUSDT"},
			{Exchange: "binance", Channel: pb.Channel_KLINE, Symbol: "BTCUSDT", Interval: "1m"},
			{Exchange: "binance", Channel: pb.Channel_BOOK, Symbol: "BTCUSDT", Depth: "5"},
		},
	}

	for {
		ctx := context.Background()
		stream, err := h.market.Subscribe(ctx, req)
		if err != nil {
			log.Printf("marketdata subscribe failed: %v, retrying in %v", err, backoff)
			time.Sleep(backoff)
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

			msg, _ := json.Marshal(marketDataToJSON(md))
			h.broadcast("market", msg)
		}

		time.Sleep(backoff)
		backoff = min(backoff*2, maxBackoff)
	}
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
func (h *MarketDataHub) SubscribeUserData(ctx context.Context, userID string, containerAddr string) (chan json.RawMessage, error) {
	key := "user:" + userID

	conn, err := grpc.NewClient(containerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial user container %s: %w", containerAddr, err)
	}

	client := pb.NewUserDataServiceClient(conn)
	stream, err := client.Subscribe(ctx, &pb.UserDataRequest{Session: "binance"})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("subscribe user data: %w", err)
	}

	ch := make(chan json.RawMessage, 64)
	h.mu.Lock()
	if h.clients[key] == nil {
		h.clients[key] = make(map[chan json.RawMessage]struct{})
	}
	h.clients[key][ch] = struct{}{}
	h.mu.Unlock()

	go func() {
		defer func() {
			conn.Close()
			h.mu.Lock()
			delete(h.clients[key], ch)
			if len(h.clients[key]) == 0 {
				delete(h.clients, key)
			}
			h.mu.Unlock()
			close(ch)
		}()

		for {
			ud, err := stream.Recv()
			if err != nil {
				log.Printf("user data stream %s error: %v", userID, err)
				return
			}
			msg, _ := json.Marshal(userDataToJSON(ud))
			select {
			case ch <- msg:
			default:
			}
		}
	}()

	return ch, nil
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
	if h.conn != nil {
		h.conn.Close()
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
