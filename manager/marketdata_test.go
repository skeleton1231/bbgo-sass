package main

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	pb "github.com/c9s/bbgo/saas/manager/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestMarketDataToJSON(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Channel:  1,
		Event:    4,
		Trades: []*pb.Trade{
			{Id: "100", Price: "50000", Quantity: "0.1", Side: 0, CreatedAt: 1234567890000},
		},
	}
	result := marketDataToJSON(md)
	if result["exchange"] != "binance" {
		t.Errorf("expected binance, got %v", result["exchange"])
	}
}

func TestPriceVolumeSlice(t *testing.T) {
	pv := []*pb.PriceVolume{
		{Price: "50000", Volume: "1.0"},
		{Price: "50100", Volume: "2.0"},
	}
	result := priceVolumeSlice(pv)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestPriceVolumeSlice_Empty(t *testing.T) {
	result := priceVolumeSlice(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestBroadcast_DropsWhenFull(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{})}
	ch := make(chan json.RawMessage, 1)
	hub.clients["market"] = map[chan json.RawMessage]struct{}{ch: {}}
	ch <- json.RawMessage(`{}`)
	hub.broadcast("market", json.RawMessage(`{"dropped":true}`))
	select {
	case msg := <-ch:
		if string(msg) != `{}` {
			t.Errorf("expected first message, got %s", msg)
		}
	default:
		t.Fatal("expected message")
	}
}

func TestChannelPb(t *testing.T) {
	if channelPb("trade") != pb.Channel_TRADE || channelPb("kline") != pb.Channel_KLINE {
		t.Error("unexpected channel mapping")
	}
}

func TestHubRedial_NilConn(t *testing.T) {
	hub := &MarketDataHub{conn: nil, clients: make(map[string]map[chan json.RawMessage]struct{}), userPool: make(map[string]*pooledConn)}
	hub.redial()
}

func TestHubClose_NilConn(t *testing.T) {
	hub := &MarketDataHub{conn: nil, userPool: make(map[string]*pooledConn)}
	hub.Close()
}

func TestHubSubscribeUnsubscribe(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{}), userPool: make(map[string]*pooledConn)}
	ch, err := hub.SubscribeMarket(nil)
	if err != nil {
		t.Fatal(err)
	}
	hub.Unsubscribe("market", ch)
	hub.mu.RLock()
	count := len(hub.clients["market"])
	hub.mu.RUnlock()
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestConnPool_SharedAcrossClients(t *testing.T) {
	var dialCount atomic.Int32
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		dialFn: func(addr string) (*grpc.ClientConn, error) {
			dialCount.Add(1)
			return grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
	}

	conn1, err := hub.getOrDial("bbgo-user1:9090")
	if err != nil {
		t.Fatal(err)
	}
	conn2, err := hub.getOrDial("bbgo-user1:9090")
	if err != nil {
		t.Fatal(err)
	}
	if dialCount.Load() != 1 {
		t.Fatalf("expected 1 dial, got %d", dialCount.Load())
	}
	if conn1 != conn2 {
		t.Fatal("expected same connection for same address")
	}
}

func TestConnPool_DifferentAddresses(t *testing.T) {
	var dialCount atomic.Int32
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		dialFn: func(addr string) (*grpc.ClientConn, error) {
			dialCount.Add(1)
			return grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
	}

	conn1, _ := hub.getOrDial("bbgo-user1:9090")
	conn2, _ := hub.getOrDial("bbgo-user2:9090")
	if dialCount.Load() != 2 {
		t.Fatalf("expected 2 dials, got %d", dialCount.Load())
	}
	if conn1 == conn2 {
		t.Fatal("different addresses should have different connections")
	}
}

func TestConnPool_RefcountRelease(t *testing.T) {
	var dialCount atomic.Int32
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		dialFn: func(addr string) (*grpc.ClientConn, error) {
			dialCount.Add(1)
			return grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
	}

	hub.getOrDial("bbgo-user1:9090")
	hub.getOrDial("bbgo-user1:9090")

	hub.releaseConn("bbgo-user1:9090")
	hub.mu.RLock()
	_, exists := hub.userPool["bbgo-user1:9090"]
	hub.mu.RUnlock()
	if !exists {
		t.Fatal("should exist after first release")
	}

	hub.releaseConn("bbgo-user1:9090")
	hub.mu.RLock()
	_, exists = hub.userPool["bbgo-user1:9090"]
	hub.mu.RUnlock()
	if exists {
		t.Fatal("should be removed after last release")
	}

	hub.getOrDial("bbgo-user1:9090")
	if dialCount.Load() != 2 {
		t.Fatalf("expected re-dial, got %d", dialCount.Load())
	}
}

func TestConnPool_ConcurrentAccess(t *testing.T) {
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		dialFn: func(addr string) (*grpc.ClientConn, error) {
			return grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := hub.getOrDial("bbgo-user1:9090")
			if err != nil {
				t.Errorf("dial error: %v", err)
			}
		}()
	}
	wg.Wait()

	hub.mu.RLock()
	pc := hub.userPool["bbgo-user1:9090"]
	hub.mu.RUnlock()
	if pc == nil {
		t.Fatal("expected pool entry")
	}
	if pc.ref != 20 {
		t.Fatalf("expected ref=20, got %d", pc.ref)
	}
}
