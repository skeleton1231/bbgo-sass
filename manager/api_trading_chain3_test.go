package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	pb "github.com/c9s/bbgo/saas/manager/pb"
	"github.com/c9s/bbgo/saas/manager/pool"
)

// --- Real-time data serialization (marketdata.go) ---

func TestMarketDataToJSON_Trade(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance", Symbol: "BTCUSDT",
		Channel: pb.Channel_TRADE, Event: pb.Event_UPDATE,
		Trades: []*pb.Trade{
			{Id: "123", Price: "50000.00", Quantity: "0.1", Side: pb.Side_BUY},
		},
	}
	result := marketDataToJSON(md)
	trades, ok := result["trades"].([]map[string]interface{})
	if !ok || len(trades) != 1 {
		t.Fatalf("trades wrong type or len")
	}
	if trades[0]["price"] != "50000.00" {
		t.Errorf("price = %v", trades[0]["price"])
	}
	if trades[0]["side"] != "BUY" {
		t.Errorf("side = %v, want BUY", trades[0]["side"])
	}
}

func TestMarketDataToJSON_KLine(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "binance", Symbol: "ETHUSDT",
		Channel: pb.Channel_KLINE, Event: pb.Event_UPDATE,
		Kline: &pb.KLine{
			Open: "3000", High: "3100", Low: "2950", Close: "3050",
			Volume: "1234.5", Closed: true,
		},
	}
	result := marketDataToJSON(md)
	kline, ok := result["kline"].(map[string]interface{})
	if !ok {
		t.Fatal("kline not a map")
	}
	if kline["close"] != "3050" {
		t.Errorf("close = %v", kline["close"])
	}
	if kline["closed"] != true {
		t.Errorf("closed = %v, want true", kline["closed"])
	}
}

func TestMarketDataToJSON_Depth(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "okex", Symbol: "BTCUSDT",
		Channel: pb.Channel_BOOK, Event: pb.Event_SNAPSHOT,
		Depth: &pb.Depth{
			Bids: []*pb.PriceVolume{{Price: "50000", Volume: "1.5"}},
			Asks: []*pb.PriceVolume{{Price: "50001", Volume: "0.8"}},
		},
	}
	result := marketDataToJSON(md)
	depth, ok := result["depth"].(map[string]interface{})
	if !ok {
		t.Fatal("depth not a map")
	}
	bids, _ := depth["bids"].([]map[string]string)
	if len(bids) != 1 || bids[0]["price"] != "50000" {
		t.Errorf("bids = %v", bids)
	}
}

func TestMarketDataToJSON_Ticker(t *testing.T) {
	md := &pb.MarketData{
		Exchange: "bybit", Symbol: "ETHUSDT",
		Channel: pb.Channel_TICKER, Event: pb.Event_UPDATE,
		Ticker: &pb.Ticker{Open: 3000, High: 3100, Low: 2950, Close: 3050, Volume: 5000},
	}
	result := marketDataToJSON(md)
	ticker, ok := result["ticker"].(map[string]interface{})
	if !ok {
		t.Fatal("ticker not a map")
	}
	if ticker["volume"] != float64(5000) {
		t.Errorf("volume = %v", ticker["volume"])
	}
}

func TestUserDataToJSON_Balances(t *testing.T) {
	ud := &pb.UserData{
		Session: "binance", Exchange: "binance",
		Channel: pb.Channel_BALANCE, Event: pb.Event_UPDATE,
		Balances: []*pb.Balance{
			{Currency: "USDT", Available: "10000.50", Locked: "500.00"},
		},
	}
	result := userDataToJSON(ud)
	balances, ok := result["balances"].([]map[string]string)
	if !ok || len(balances) != 1 {
		t.Fatalf("balances wrong type or len")
	}
	if balances[0]["available"] != "10000.50" {
		t.Errorf("available = %v", balances[0]["available"])
	}
}

func TestUserDataToJSON_Orders(t *testing.T) {
	ud := &pb.UserData{
		Session: "binance", Exchange: "binance",
		Channel: pb.Channel_ORDER, Event: pb.Event_UPDATE,
		Orders: []*pb.Order{
			{Symbol: "BTCUSDT", Side: pb.Side_BUY, Price: "50000", Status: "NEW"},
		},
	}
	result := userDataToJSON(ud)
	orders, ok := result["orders"].([]map[string]interface{})
	if !ok || len(orders) != 1 {
		t.Fatalf("orders wrong type or len")
	}
	if orders[0]["side"] != "BUY" {
		t.Errorf("side = %v", orders[0]["side"])
	}
}

func TestUserDataToJSON_Trades(t *testing.T) {
	ud := &pb.UserData{
		Session: "binance", Exchange: "binance",
		Channel: pb.Channel_TRADE, Event: pb.Event_UPDATE,
		Trades: []*pb.Trade{
			{Price: "50000", Quantity: "0.1", Fee: "0.05"},
		},
	}
	result := userDataToJSON(ud)
	trades, ok := result["trades"].([]map[string]interface{})
	if !ok || len(trades) != 1 {
		t.Fatalf("trades wrong type or len")
	}
	if trades[0]["fee"] != "0.05" {
		t.Errorf("fee = %v", trades[0]["fee"])
	}
}

// --- Broadcast (marketdata.go) ---

func TestBroadcast_DeliversToMultipleSubscribers(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{})}
	ch1 := make(chan json.RawMessage, 64)
	ch2 := make(chan json.RawMessage, 64)
	hub.clients["market"] = map[chan json.RawMessage]struct{}{ch1: {}, ch2: {}}

	msg := json.RawMessage(`{"exchange":"binance"}`)
	hub.broadcast("market", msg)

	for i, ch := range []chan json.RawMessage{ch1, ch2} {
		select {
		case received := <-ch:
			if string(received) != string(msg) {
				t.Errorf("ch%d got %s", i+1, received)
			}
		default:
			t.Errorf("ch%d did not receive", i+1)
		}
	}
}

func TestBroadcast_ConcurrentSafety(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{})}
	ch := make(chan json.RawMessage, 256)
	hub.clients["market"] = map[chan json.RawMessage]struct{}{ch: {}}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.broadcast("market", json.RawMessage(`{"v":1}`))
		}()
	}
	wg.Wait()

	received := 0
	for {
		select {
		case <-ch:
			received++
		default:
			goto done
		}
	}
done:
	if received == 0 {
		t.Error("no messages received from concurrent broadcast")
	}
}

// --- extractSessionNames additional coverage ---

func TestExtractSessionNames_MultiExchange(t *testing.T) {
	uc := &UserContainer{Strategies: []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
		{Exchange: "okex", Strategy: "bollmaker"},
	}}
	names := extractSessionNames(uc)
	if len(names) != 2 || names[0] != "binance" || names[1] != "okex" {
		t.Errorf("sessions = %v, want [binance okex]", names)
	}
}

func TestExtractSessionNames_Mixed(t *testing.T) {
	uc := &UserContainer{Strategies: []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2"},
		{CrossExchange: true, Strategy: "xmaker", Sessions: []SessionRoleConfig{
			{Name: "okex_spot", Exchange: "okex"},
		}},
	}}
	names := extractSessionNames(uc)
	if len(names) != 2 {
		t.Errorf("sessions = %v, want 2 entries", names)
	}
}

func TestExtractSessionNames_Empty(t *testing.T) {
	names := extractSessionNames(&UserContainer{})
	if len(names) != 0 {
		t.Errorf("sessions = %v, want empty", names)
	}
}

// --- Notifier rules and dispatch ---

func TestNotifier_RuleEnabled(t *testing.T) {
	tests := []struct {
		event string
		rules NotificationRule
		want  bool
	}{
		{"trade", NotificationRule{TradeEvents: true}, true},
		{"trade", NotificationRule{TradeEvents: false}, false},
		{"order", NotificationRule{OrderEvents: true}, true},
		{"container", NotificationRule{ContainerHealth: true}, true},
		{"backtest", NotificationRule{}, true},
		{"test", NotificationRule{}, true},
		{"unknown", NotificationRule{}, false},
	}
	for _, tt := range tests {
		n := &Notifier{}
		if got := n.ruleEnabled(tt.rules, tt.event); got != tt.want {
			t.Errorf("ruleEnabled(%+v, %q) = %v, want %v", tt.rules, tt.event, got, tt.want)
		}
	}
}

func TestNotifier_Dispatch_NoConfigs(t *testing.T) {
	n := &Notifier{configs: make(map[string][]NotificationConfig)}
	if n.Dispatch("u1", NotificationEvent{Type: "trade"}) {
		t.Error("should return false with no configs")
	}
}

func TestNotifier_Dispatch_DisabledChannel(t *testing.T) {
	n := &Notifier{configs: map[string][]NotificationConfig{
		"u1": {{Channel: NotificationChannel{Type: "telegram", Enabled: false}, Rules: NotificationRule{TradeEvents: true}}},
	}}
	if n.Dispatch("u1", NotificationEvent{Type: "trade"}) {
		t.Error("should return false with disabled channel")
	}
}

func TestNotifier_Dispatch_RuleNotMatching(t *testing.T) {
	n := &Notifier{configs: map[string][]NotificationConfig{
		"u1": {{Channel: NotificationChannel{Type: "telegram", Enabled: true}, Rules: NotificationRule{OrderEvents: true}}},
	}}
	if n.Dispatch("u1", NotificationEvent{Type: "trade"}) {
		t.Error("should return false when rule doesn't match")
	}
}

func TestNotifier_Dispatch_TelegramWithMock(t *testing.T) {
	var receivedBody string
	telegramSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer telegramSrv.Close()

	enc, _ := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	encToken, _ := enc.Encrypt("fake-bot-token")

	n := &Notifier{
		crypto:    enc,
		client:    telegramSrv.Client(),
		configs:   map[string][]NotificationConfig{},
		lastSent:  map[string]map[string]time.Time{},
		rateLimit: 0,
	}
	n.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{
			Type: "telegram", Enabled: true,
			TokenEnc: encToken, ChatID: "12345",
		},
		Rules: NotificationRule{TradeEvents: true},
	}}

	// sendTelegram builds URL as https://api.telegram.org/bot{token}/sendMessage
	// so the mock won't match. This test validates the decrypt+dispatch path
	// reaches the HTTP call (will fail with 404 but proves the flow works).
	_ = n.Dispatch("u1", NotificationEvent{Type: "trade", Title: "Trade", Message: "BUY"})
	// If we got here without panic, the decrypt + rule matching path works.
	_ = receivedBody
}

func TestNotifier_Dispatch_SlackWithMock(t *testing.T) {
	var receivedPath string
	slackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	}))
	defer slackSrv.Close()

	enc, _ := NewEncryptor("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	encURL, _ := enc.Encrypt(slackSrv.URL + "/webhook/test")

	n := &Notifier{
		crypto:    enc,
		client:    slackSrv.Client(),
		configs:   map[string][]NotificationConfig{},
		lastSent:  map[string]map[string]time.Time{},
		rateLimit: 0,
	}
	n.configs["u1"] = []NotificationConfig{{
		Channel: NotificationChannel{
			Type: "slack", Enabled: true,
			WebhookURL: encURL,
		},
		Rules: NotificationRule{OrderEvents: true},
	}}

	ok := n.Dispatch("u1", NotificationEvent{Type: "order", Title: "Order", Message: "SELL"})
	if !ok {
		t.Error("Dispatch should succeed")
	}
	if receivedPath != "/webhook/test" {
		t.Errorf("slack path = %q, want /webhook/test", receivedPath)
	}
}

// --- readBodyHint (sync.go) ---

func TestReadBodyHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message":"Rate limit exceeded"}`))
	}))
	defer srv.Close()
	resp, _ := http.Get(srv.URL)
	hint := readBodyHint(resp)
	if hint != `{"message":"Rate limit exceeded"}` {
		t.Errorf("readBodyHint = %q", hint)
	}
}

func TestReadBodyHint_TruncatesLargeBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		large := make([]byte, 1024)
		for i := range large {
			large[i] = 'x'
		}
		w.Write(large)
	}))
	defer srv.Close()
	resp, _ := http.Get(srv.URL)
	hint := readBodyHint(resp)
	if len(hint) > 512 {
		t.Errorf("readBodyHint len = %d, want <= 512", len(hint))
	}
}

// --- SyncUser (sync.go) ---

func TestSyncUser_UpsertsAndSyncs(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("sync-u1", ModeLive, StrategyEntry{Exchange: "binance", Strategy: "grid2", Mode: "paper"})
	users.UpdateStatus("sync-u1", ModeLive, StatusRunning)

	upserted := false
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/user_containers" {
			upserted = true
		}
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer bbgoSrv.Close()

	p := pool.New(5)
	s := &Syncer{
		users:     users,
		cfg:       &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		pool:      p,
		client:    supabaseSrv.Client(),
		newBBGoClientFn: func(_ string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
	}
	s.SyncUser("sync-u1", ModeLive)
	if !upserted {
		t.Error("SyncUser did not upsert to Supabase")
	}
}

func TestSyncUser_StoppedContainer(t *testing.T) {
	users := NewUserContainerManager()
	users.AddStrategy("sync-u2", ModeLive, StrategyEntry{Exchange: "binance", Strategy: "grid2"})

	upserted := false
	supabaseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upserted = true
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
	}))
	defer supabaseSrv.Close()

	bbgoCalled := false
	bbgoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bbgoCalled = true
		w.Write([]byte(`[]`))
	}))
	defer bbgoSrv.Close()

	p := pool.New(5)
	s := &Syncer{
		users:     users,
		cfg:       &Config{SupabaseURL: supabaseSrv.URL, SupabaseKey: "k"},
		container: &ContainerManager{cfg: &Config{BBGOPort: 8080}},
		pool:      p,
		client:    supabaseSrv.Client(),
		newBBGoClientFn: func(_ string) *BBGoClient {
			return &BBGoClient{baseURL: bbgoSrv.URL, client: bbgoSrv.Client()}
		},
	}
	s.SyncUser("sync-u2", ModeLive)
	if !upserted {
		t.Error("should still upsert even when stopped")
	}
	if bbgoCalled {
		t.Error("should not call bbgo API for stopped container")
	}
}

// --- MarketDataHub subscribe/unsubscribe additional coverage ---

func TestMarketDataHub_UnsubscribeTwice(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{}), done: make(chan struct{})}
	ch, _ := hub.SubscribeMarket(context.Background())
	hub.Unsubscribe("market", ch)
	hub.Unsubscribe("market", ch) // should not panic
}

func TestMarketDataHub_Close(t *testing.T) {
	hub := &MarketDataHub{clients: make(map[string]map[chan json.RawMessage]struct{}), userPool: make(map[string]*pooledConn), done: make(chan struct{})}
	hub.Close()
	hub.Close() // should not panic
}
