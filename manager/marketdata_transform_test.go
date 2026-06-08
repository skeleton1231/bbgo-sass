package main

import (
	"encoding/json"
	"testing"

	pb "github.com/c9s/bbgo/saas/manager/pb"
)

func TestMarketDataToJSON_AllFields(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Channel:  pb.Channel_KLINE,
		Event:    pb.Event_UPDATE,
		Depth: &pb.Depth{
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Bids:     []*pb.PriceVolume{{Price: "50000", Volume: "1.5"}},
			Asks:     []*pb.PriceVolume{{Price: "50100", Volume: "2.0"}},
		},
		Kline: &pb.KLine{
			Open:      "49900",
			High:      "50200",
			Low:       "49800",
			Close:     "50100",
			Volume:    "123.45",
			StartTime: 1700000000000,
			Closed:    true,
		},
		Ticker: &pb.Ticker{
			Open:   50000,
			High:   50500,
			Low:    49500,
			Close:  50100,
			Volume: 1000,
		},
		Trades: []*pb.Trade{
			{Id: "1", Price: "50000", Quantity: "0.5", Side: pb.Side_BUY, CreatedAt: 1700000000000},
			{Id: "2", Price: "50100", Quantity: "0.3", Side: pb.Side_SELL, CreatedAt: 1700000001000},
		},
	}

	result := marketDataToJSON(md)

	if result["exchange"] != "binance" {
		t.Errorf("expected binance, got %v", result["exchange"])
	}
	if result["symbol"] != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %v", result["symbol"])
	}

	depth, ok := result["depth"].(map[string]any)
	if !ok {
		t.Fatal("expected depth map")
	}
	bids, ok := depth["bids"].([]map[string]any)
	if !ok || len(bids) != 1 {
		t.Fatalf("expected 1 bid, got %v", depth["bids"])
	}
	if bids[0]["price"] != "50000" {
		t.Errorf("expected bid price 50000, got %s", bids[0]["price"])
	}

	kline, ok := result["kline"].(map[string]any)
	if !ok {
		t.Fatal("expected kline map")
	}
	if kline["open"] != "49900" {
		t.Errorf("expected kline open 49900, got %v", kline["open"])
	}

	ticker, ok := result["ticker"].(map[string]any)
	if !ok {
		t.Fatal("expected ticker map")
	}
	if ticker["close"] != float64(50100) {
		t.Errorf("expected ticker close 50100, got %v", ticker["close"])
	}

	trades, ok := result["trades"].([]map[string]any)
	if !ok || len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %v", result["trades"])
	}
	if trades[0]["side"] != "BUY" {
		t.Errorf("expected BUY, got %v", trades[0]["side"])
	}
}

func TestMarketDataToJSON_EmptyFields(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance",
		Symbol:   "ETHUSDT",
		Channel:  pb.Channel_TRADE,
		Event:    pb.Event_SNAPSHOT,
	}
	result := marketDataToJSON(md)

	if _, exists := result["depth"]; exists {
		t.Error("depth should not exist for nil Depth")
	}
	if _, exists := result["kline"]; exists {
		t.Error("kline should not exist for nil Kline")
	}
	if _, exists := result["ticker"]; exists {
		t.Error("ticker should not exist for nil Ticker")
	}
	if _, exists := result["trades"]; exists {
		t.Error("trades should not exist for empty Trades")
	}
}

func TestUserDataToJSON_AllFields(t *testing.T) {
	ud := &pb.UserData{
		Session:  "binance",
		Exchange: "binance",
		Channel:  pb.Channel_BALANCE,
		Event:    pb.Event_SNAPSHOT,
		Balances: []*pb.Balance{
			{Currency: "BTC", Available: "1.5", Locked: "0.5"},
			{Currency: "USDT", Available: "10000", Locked: "2000"},
		},
		Orders: []*pb.Order{
			{Id: "100", Symbol: "BTCUSDT", Side: pb.Side_BUY, Price: "50000", Quantity: "0.1", ExecutedQuantity: "0.05", Status: "NEW"},
		},
		Trades: []*pb.Trade{
			{Id: "200", Price: "49900", Quantity: "0.05", Side: pb.Side_SELL, Fee: "0.001"},
		},
	}

	result := userDataToJSON(ud)

	if result["session"] != "binance" {
		t.Errorf("expected binance, got %v", result["session"])
	}

	balances, ok := result["balances"].([]map[string]any)
	if !ok || len(balances) != 2 {
		t.Fatalf("expected 2 balances, got %v", result["balances"])
	}
	if balances[0]["currency"] != "BTC" {
		t.Errorf("expected BTC, got %s", balances[0]["currency"])
	}
	if balances[0]["available"] != "1.5" {
		t.Errorf("expected 1.5, got %s", balances[0]["available"])
	}

	orders, ok := result["orders"].([]map[string]any)
	if !ok || len(orders) != 1 {
		t.Fatalf("expected 1 order, got %v", result["orders"])
	}
	if orders[0]["symbol"] != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %v", orders[0]["symbol"])
	}
		if orders[0]["orderType"] != "MARKET" {
			t.Errorf("expected MARKET (zero value), got %v", orders[0]["orderType"])
		}

	trades, ok := result["trades"].([]map[string]any)
	if !ok || len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %v", result["trades"])
	}
	if trades[0]["fee"] != "0.001" {
		t.Errorf("expected fee 0.001, got %v", trades[0]["fee"])
	}
}

func TestUserDataToJSON_Empty(t *testing.T) {
	ud := &pb.UserData{
		Session:  "binance",
		Exchange: "binance",
		Channel:  pb.Channel_ORDER,
		Event:    pb.Event_UPDATE,
	}
	result := userDataToJSON(ud)

	if _, exists := result["balances"]; exists {
		t.Error("balances should not exist when empty")
	}
	if _, exists := result["orders"]; exists {
		t.Error("orders should not exist when empty")
	}
	if _, exists := result["trades"]; exists {
		t.Error("trades should not exist when empty")
	}
}

func TestChannelPb_AllMappings(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.Channel
	}{
		{"trade", pb.Channel_TRADE},
		{"kline", pb.Channel_KLINE},
		{"book", pb.Channel_BOOK},
		{"unknown", pb.Channel_TRADE},
		{"", pb.Channel_TRADE},
	}
	for _, tt := range tests {
		got := channelPb(tt.input)
		if got != tt.expected {
			t.Errorf("channelPb(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestMarketDataToJSON_RoundTrip(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Channel:  pb.Channel_KLINE,
		Event:    pb.Event_UPDATE,
		Kline: &pb.KLine{
			Open: "100", High: "110", Low: "90", Close: "105",
			Volume: "500", StartTime: 1700000000000, Closed: true,
		},
	}

	converted := marketDataToJSON(md)
	data, err := json.Marshal(converted)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed["exchange"] != "binance" {
		t.Errorf("round-trip: expected binance, got %v", parsed["exchange"])
	}
}

func TestBroadcast_MultipleClients(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{})}
	ch1 := make(chan json.RawMessage, 4)
	ch2 := make(chan json.RawMessage, 4)
	hub.clients["market"] = map[chan json.RawMessage]struct{}{
		ch1: {},
		ch2: {},
	}

	msg := json.RawMessage(`{"test":true}`)
	hub.broadcast("market", msg)

	for i, ch := range []chan json.RawMessage{ch1, ch2} {
		select {
		case received := <-ch:
			if string(received) != `{"test":true}` {
				t.Errorf("client %d: expected {\"test\":true}, got %s", i, received)
			}
		default:
			t.Errorf("client %d: expected message", i)
		}
	}
}

func TestBroadcast_NonexistentKey(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{})}
	hub.broadcast("nonexistent", json.RawMessage(`{}`))
}

func TestHubClose_DoubleClose(t *testing.T) {
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		done:     make(chan struct{}),
	}
	hub.Close()
	hub.Close()
}

func TestHubClose_WithUserPool(t *testing.T) {
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
		done:     make(chan struct{}),
	}
	hub.Close()
	if len(hub.userPool) != 0 {
		t.Error("user pool should be empty after close")
	}
}

func TestSubscribeUnsubscribe_Cleanup(t *testing.T) {
	hub := &MarketDataHub{
		clients:  make(map[string]map[chan json.RawMessage]struct{}),
		userPool: make(map[string]*pooledConn),
	}

	ch1, _ := hub.SubscribeMarket(nil)
	ch2, _ := hub.SubscribeMarket(nil)

	hub.mu.RLock()
	count := len(hub.clients["market"])
	hub.mu.RUnlock()
	if count != 2 {
		t.Fatalf("expected 2 subscribers, got %d", count)
	}

	hub.Unsubscribe("market", ch1)
	hub.mu.RLock()
	count = len(hub.clients["market"])
	hub.mu.RUnlock()
	if count != 1 {
		t.Fatalf("expected 1 subscriber after unsubscribe, got %d", count)
	}

	hub.Unsubscribe("market", ch2)
	hub.mu.RLock()
	_, exists := hub.clients["market"]
	hub.mu.RUnlock()
	if exists {
		t.Error("market key should be removed after all unsubscribes")
	}

	hub.Unsubscribe("market", ch2)
}
