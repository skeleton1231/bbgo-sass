package main

import (
	"encoding/json"
	"testing"

	pb "github.com/c9s/bbgo/saas/manager/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestExtractSessionNames_SingleExchange(t *testing.T) {
	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid"},
		},
	}
	sessions := extractSessionNames(uc)
	if len(sessions) != 1 || sessions[0] != "binance" {
		t.Fatalf("expected [binance], got %v", sessions)
	}
}

func TestExtractSessionNames_CrossExchange(t *testing.T) {
	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "binance", Exchange: "binance"},
					{Name: "okex", Exchange: "okex"},
				},
			},
		},
	}
	sessions := extractSessionNames(uc)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %v", len(sessions), sessions)
	}
	if sessions[0] != "binance" || sessions[1] != "okex" {
		t.Fatalf("expected [binance okex], got %v", sessions)
	}
}

func TestExtractSessionNames_MixedStrategies(t *testing.T) {
	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid"},
			{
				CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "bybit", Exchange: "bybit"},
					{Name: "binance", Exchange: "binance"},
				},
			},
		},
	}
	sessions := extractSessionNames(uc)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 unique sessions, got %d: %v", len(sessions), sessions)
	}
	if sessions[0] != "binance" || sessions[1] != "bybit" {
		t.Fatalf("expected [binance bybit], got %v", sessions)
	}
}

func TestExtractSessionNames_Deduplicates(t *testing.T) {
	uc := &UserContainer{
		UserID: "user-1",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid"},
			{Exchange: "binance", Strategy: "xmaker"},
		},
	}
	sessions := extractSessionNames(uc)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 unique session, got %d: %v", len(sessions), sessions)
	}
}

func TestExtractSessionNames_Empty(t *testing.T) {
	uc := &UserContainer{
		UserID:     "user-1",
		Strategies: []StrategyEntry{},
	}
	sessions := extractSessionNames(uc)
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

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
	if result["symbol"] != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %v", result["symbol"])
	}
	trades, ok := result["trades"].([]map[string]interface{})
	if !ok || len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %v", result["trades"])
	}
	if trades[0]["price"] != "50000" {
		t.Errorf("expected price 50000, got %v", trades[0]["price"])
	}
}

func TestMarketDataToJSON_WithKline(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance",
		Symbol:   "ETHUSDT",
		Channel:  3,
		Kline:    &pb.KLine{Open: "3000", High: "3100", Low: "2950", Close: "3050", Volume: "100", Closed: true},
	}
	result := marketDataToJSON(md)
	kline, ok := result["kline"].(map[string]interface{})
	if !ok {
		t.Fatal("expected kline map")
	}
	if kline["close"] != "3050" {
		t.Errorf("expected close 3050, got %v", kline["close"])
	}
	if kline["closed"] != true {
		t.Errorf("expected closed=true, got %v", kline["closed"])
	}
}

func TestMarketDataToJSON_WithDepth(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Channel:  0,
		Depth: &pb.Depth{
			Bids: []*pb.PriceVolume{{Price: "50000", Volume: "1.5"}},
			Asks: []*pb.PriceVolume{{Price: "50100", Volume: "2.0"}},
		},
	}
	result := marketDataToJSON(md)
	depth, ok := result["depth"].(map[string]interface{})
	if !ok {
		t.Fatal("expected depth map")
	}
	bids := depth["bids"].([]map[string]string)
	if len(bids) != 1 || bids[0]["price"] != "50000" {
		t.Errorf("unexpected bids: %v", bids)
	}
}

func TestUserDataToJSON(t *testing.T) {
	ud := &pb.UserData{
		Session:  "binance",
		Exchange: "binance",
		Balances: []*pb.Balance{
			{Currency: "BTC", Available: "1.5", Locked: "0.5"},
		},
		Orders: []*pb.Order{
			{Id: "100", Symbol: "BTCUSDT", Side: 0, Price: "50000", Quantity: "0.1", Status: "NEW"},
		},
		Trades: []*pb.Trade{
			{Id: "200", Price: "50000", Quantity: "0.1", Side: 1, Fee: "0.001"},
		},
	}
	result := userDataToJSON(ud)

	if result["session"] != "binance" {
		t.Errorf("expected session binance, got %v", result["session"])
	}
	balances := result["balances"].([]map[string]string)
	if len(balances) != 1 || balances[0]["currency"] != "BTC" {
		t.Errorf("unexpected balances: %v", balances)
	}
	orders := result["orders"].([]map[string]interface{})
	if len(orders) != 1 || orders[0]["status"] != "NEW" {
		t.Errorf("unexpected orders: %v", orders)
	}
	trades := result["trades"].([]map[string]interface{})
	if len(trades) != 1 || trades[0]["fee"] != "0.001" {
		t.Errorf("unexpected trades: %v", trades)
	}
}

func TestPriceVolumeSlice(t *testing.T) {
	pv := []*pb.PriceVolume{
		{Price: "50000", Volume: "1.0"},
		{Price: "50100", Volume: "2.0"},
	}
	result := priceVolumeSlice(pv)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0]["price"] != "50000" || result[1]["volume"] != "2.0" {
		t.Errorf("unexpected: %v", result)
	}
}

func TestPriceVolumeSlice_Empty(t *testing.T) {
	result := priceVolumeSlice(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(result))
	}
}

func TestBroadcast_DropsWhenFull(t *testing.T) {
	hub := &MarketDataHub{
		clients: make(map[string]map[chan json.RawMessage]struct{}),
	}
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
		t.Fatal("expected message in channel")
	}
}

func TestChannelPb(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.Channel
	}{
		{"trade", pb.Channel_TRADE},
		{"kline", pb.Channel_KLINE},
		{"book", pb.Channel_BOOK},
		{"unknown", pb.Channel_TRADE},
	}
	for _, tt := range tests {
		got := channelPb(tt.input)
		if got != tt.expected {
			t.Errorf("channelPb(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestHubRedial_NilConn(t *testing.T) {
	hub := &MarketDataHub{conn: nil, clients: make(map[string]map[chan json.RawMessage]struct{})}
	hub.redial()
}

func TestHubRedial_ConnReplacedAtomically(t *testing.T) {
	hub := &MarketDataHub{
		addr:    "passthrough:///localhost:1",
		clients: make(map[string]map[chan json.RawMessage]struct{}),
	}

	conn, err := grpc.NewClient(hub.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("grpc client: %v", err)
	}
	hub.conn = conn

	hub.redial()

	hub.mu.RLock()
	newConn := hub.conn
	newClient := hub.market
	hub.mu.RUnlock()

	if newConn == nil {
		t.Fatal("expected conn to be replaced, got nil")
	}
	if newClient == nil {
		t.Fatal("expected market client to be replaced, got nil")
	}
}

func TestHubClose_NilConn(t *testing.T) {
	hub := &MarketDataHub{conn: nil}
	hub.Close()
}

func TestHubSubscribeUnsubscribe(t *testing.T) {
	hub := &MarketDataHub{
		clients: make(map[string]map[chan json.RawMessage]struct{}),
	}

	ch, err := hub.SubscribeMarket(nil)
	if err != nil {
		t.Fatalf("SubscribeMarket error: %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	hub.mu.RLock()
	count := len(hub.clients["market"])
	hub.mu.RUnlock()
	if count != 1 {
		t.Fatalf("expected 1 subscriber, got %d", count)
	}

	hub.Unsubscribe("market", ch)
	hub.mu.RLock()
	count = len(hub.clients["market"])
	hub.mu.RUnlock()
	if count != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", count)
	}
}
