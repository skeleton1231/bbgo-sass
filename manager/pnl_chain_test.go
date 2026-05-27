package main

import (
	"math"
	"strings"
	"testing"
)

// --- PnL calculation: basic BUY then SELL (realized profit) ---

func TestPnL_BuyThenSell_Profit(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0.001", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0.001", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)

	if len(report.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(report.Symbols))
	}
	s := report.Symbols[0]
	if math.Abs(s.RealizedPnL-5000) > 0.01 {
		t.Errorf("expected realized PnL ~5000, got %.2f", s.RealizedPnL)
	}
	if s.TradeCount != 2 {
		t.Errorf("expected 2 trades, got %d", s.TradeCount)
	}
	if s.WinningTrades != 1 {
		t.Errorf("expected 1 winning trade, got %d", s.WinningTrades)
	}
	if s.OpenPosition != 0 {
		t.Errorf("expected 0 open position, got %.8f", s.OpenPosition)
	}
}

// --- PnL: BUY then SELL at a loss ---

func TestPnL_BuyThenSell_Loss(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "45000", Quantity: "1", Fee: "0", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)

	s := report.Symbols[0]
	if s.RealizedPnL > 0 {
		t.Errorf("expected negative PnL, got %.2f", s.RealizedPnL)
	}
	if s.LosingTrades != 1 {
		t.Errorf("expected 1 losing trade, got %d", s.LosingTrades)
	}
}

// --- PnL: only BUYs (no sells -> open position) ---

func TestPnL_OnlyBuys_OpenPosition(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0.5", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "52000", Quantity: "0.5", Fee: "0", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)

	s := report.Symbols[0]
	if s.RealizedPnL != 0 {
		t.Errorf("expected 0 realized PnL (no sells), got %.2f", s.RealizedPnL)
	}
	if math.Abs(s.OpenPosition-1.0) > 1e-8 {
		t.Errorf("expected open position 1.0, got %.8f", s.OpenPosition)
	}
	avgCost := s.OpenPositionCost / s.OpenPosition
	if math.Abs(avgCost-51000) > 0.01 {
		t.Errorf("expected avg cost 51000, got %.2f", avgCost)
	}
}

// --- PnL: FIFO partial close (partial sell of layered buys) ---

func TestPnL_FIFO_PartialSell(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "60000", Quantity: "1", Fee: "0", TradedAt: "2024-01-02"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0", TradedAt: "2024-01-03"},
	}
	report := calculatePnL(trades)

	s := report.Symbols[0]
	// FIFO: first sell matches first buy (50000 * 1). PnL = 55000 - 50000 = 5000
	if math.Abs(s.RealizedPnL-5000) > 0.01 {
		t.Errorf("expected FIFO PnL 5000, got %.2f", s.RealizedPnL)
	}
	if math.Abs(s.OpenPosition-1.0) > 1e-8 {
		t.Errorf("expected open position 1.0, got %.8f", s.OpenPosition)
	}
}

// --- PnL: multiple symbols ---

func TestPnL_MultipleSymbols(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "10", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "10", TradedAt: "2024-01-02"},
		{Symbol: "ETHUSDT", Side: "BUY", Price: "3000", Quantity: "5", Fee: "5", TradedAt: "2024-01-01"},
		{Symbol: "ETHUSDT", Side: "SELL", Price: "3500", Quantity: "5", Fee: "5", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)

	if len(report.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(report.Symbols))
	}
	if math.Abs(report.TotalRealizedPnL-7500) > 0.01 {
		t.Errorf("expected total realized PnL 7500 (5000+2500), got %.2f", report.TotalRealizedPnL)
	}
	if report.TotalTrades != 4 {
		t.Errorf("expected 4 total trades, got %d", report.TotalTrades)
	}
	if report.WinningTrades != 2 {
		t.Errorf("expected 2 winning trades, got %d", report.WinningTrades)
	}
}

// --- PnL: zero trades ---

func TestPnL_ZeroTrades(t *testing.T) {
	report := calculatePnL(nil)
	if report.TotalTrades != 0 {
		t.Errorf("expected 0 trades, got %d", report.TotalTrades)
	}
	if len(report.Symbols) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(report.Symbols))
	}
}

// --- PnL: zero price/quantity trades are skipped ---

func TestPnL_ZeroPriceSkipped(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "0", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "0", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
	}
	report := calculatePnL(trades)

	if report.TotalTrades != 1 {
		t.Errorf("expected 1 valid trade (zero price/qty skipped), got %d", report.TotalTrades)
	}
}

// --- PnL: win rate calculation ---

func TestPnL_WinRate(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0", TradedAt: "2024-01-02"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "55000", Quantity: "1", Fee: "0", TradedAt: "2024-01-03"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-04"},
	}
	report := calculatePnL(trades)

	if math.Abs(report.WinRate-50) > 0.01 {
		t.Errorf("expected 50%% win rate, got %.2f%%", report.WinRate)
	}
}

// --- PnL: sell exceeding buys (unmatched position) ---

func TestPnL_SellExceedsBuys(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1.5", Fee: "0", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)

	s := report.Symbols[0]
	// FIFO: 1 BTC matched at 50000, remaining 0.5 uses sell price (55000) as cost basis
	// Realized = (55000 * 1.5) - (50000*1 + 55000*0.5) = 82500 - 77500 = 5000
	if math.Abs(s.RealizedPnL-5000) > 0.01 {
		t.Errorf("expected PnL 5000 with unmatched sell, got %.2f", s.RealizedPnL)
	}
}

// --- PnL: fees tracked correctly ---

func TestPnL_FeesTracked(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "25", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "27.5", TradedAt: "2024-01-02"},
	}
	report := calculatePnL(trades)

	if math.Abs(report.TotalFees-52.5) > 0.01 {
		t.Errorf("expected total fees 52.5, got %.2f", report.TotalFees)
	}
}

// --- PnL: average buy/sell price ---

func TestPnL_AvgPrices(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", TradedAt: "2024-01-01"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "60000", Quantity: "1", Fee: "0", TradedAt: "2024-01-02"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "58000", Quantity: "1", Fee: "0", TradedAt: "2024-01-03"},
	}
	report := calculatePnL(trades)

	s := report.Symbols[0]
	if math.Abs(s.AvgBuyPrice-55000) > 0.01 {
		t.Errorf("expected avg buy price 55000, got %.2f", s.AvgBuyPrice)
	}
	if math.Abs(s.AvgSellPrice-58000) > 0.01 {
		t.Errorf("expected avg sell price 58000, got %.2f", s.AvgSellPrice)
	}
}

// --- envArgs: cross-exchange credential injection ---

func TestEnvArgs_CrossExchange_InjectsMultipleCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	userID := "user-cross"

	for _, ex := range []string{"binance", "okex"} {
		key, _ := enc.Encrypt(ex + "_key")
		secret, _ := enc.Encrypt(ex + "_secret")
		creds.Upsert(ExchangeCredential{
			ID: "cred-" + ex, UserID: userID, Exchange: ex,
			APIKeyEncrypted: key, APISecretEncrypted: secret,
		})
	}

	cfg := &Config{DataDir: tmpDir, BBGOPort: 8080}
	cm := &ContainerManager{cfg: cfg, creds: creds}

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: userID,
		Strategies: []StrategyEntry{
			{
				ID: "s1", Strategy: "xmaker", CrossExchange: true,
				Sessions: []SessionRoleConfig{
					{Name: "binance", Exchange: "binance", EnvVarPrefix: "BINANCE"},
					{Name: "okex", Exchange: "okex", EnvVarPrefix: "OKEX"},
				},
			},
		},
	}

	args := cm.envArgs(uc)

	hasBinanceKey := false
	hasOkexKey := false
	for _, a := range args {
		if a == "BINANCE_API_KEY=binance_key" {
			hasBinanceKey = true
		}
		if a == "OKEX_API_KEY=okex_key" {
			hasOkexKey = true
		}
	}
	if !hasBinanceKey {
		t.Error("expected BINANCE_API_KEY in env args for cross-exchange strategy")
	}
	if !hasOkexKey {
		t.Error("expected OKEX_API_KEY in env args for cross-exchange strategy")
	}
}

// --- envArgs: passphrase injection for exchanges that need it ---

func TestEnvArgs_PassphraseInjection(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	userID := "user-okex"

	key, _ := enc.Encrypt("okex_key")
	secret, _ := enc.Encrypt("okex_secret")
	passphrase, _ := enc.Encrypt("okex_pass")
	creds.Upsert(ExchangeCredential{
		ID: "cred-okex", UserID: userID, Exchange: "okex",
		APIKeyEncrypted: key, APISecretEncrypted: secret, PassphraseEncrypted: passphrase,
	})

	cfg := &Config{DataDir: tmpDir, BBGOPort: 8080}
	cm := &ContainerManager{cfg: cfg, creds: creds}

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: userID,
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "okex", Strategy: "grid2", Mode: "live"},
		},
	}

	args := cm.envArgs(uc)

	hasPassphrase := false
	for _, a := range args {
		if a == "OKEX_PASSPHRASE=okex_pass" {
			hasPassphrase = true
		}
	}
	if !hasPassphrase {
		t.Errorf("expected OKEX_PASSPHRASE in env args, got %v", args)
	}
}

// --- envArgs: no duplicate injection for same exchange across strategies ---

func TestEnvArgs_NoDuplicateForSameExchange(t *testing.T) {
	tmpDir := t.TempDir()
	enc, _ := NewEncryptor(testEncryptionKey)
	creds := NewCredentialStore(tmpDir, enc)
	userID := "user-dup"

	key, _ := enc.Encrypt("key1")
	secret, _ := enc.Encrypt("secret1")
	creds.Upsert(ExchangeCredential{
		ID: "cred-1", UserID: userID, Exchange: "binance",
		APIKeyEncrypted: key, APISecretEncrypted: secret,
	})

	cfg := &Config{DataDir: tmpDir, BBGOPort: 8080}
	cm := &ContainerManager{cfg: cfg, creds: creds}

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: userID,
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live"},
			{ID: "s2", Exchange: "binance", Strategy: "grid2", Mode: "live", Config: []byte(`{"symbol":"ETHUSDT"}`)},
		},
	}

	args := cm.envArgs(uc)

	count := 0
	for _, a := range args {
		if a == "BINANCE_API_KEY=key1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected BINANCE_API_KEY injected exactly once, got %d times", count)
	}
}

// --- buildUserYAML: paper mode sets PAPER_TRADE in YAML environment ---

func TestBuildUserYAML_PaperEnv_PublicOnly(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModePaper,
		UserID: "user-yaml",
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "paper",
				Config: []byte(`{"symbol":"BTCUSDT","upperPrice":"60000","lowerPrice":"40000","gridNumber":10,"quantity":"0.001"}`)},
		},
	}

	yamlBytes, err := buildUserYAML(uc, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if !strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Errorf("paper mode should set PAPER_TRADE in YAML environment, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "publicOnly: true") {
		t.Errorf("paper mode without credentials should set publicOnly, got:\n%s", yamlStr)
	}
}

// --- buildUserYAML: live mode with credentials does NOT set publicOnly ---

func TestBuildUserYAML_LiveEnv_NoPublicOnly(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "user-yaml",
		Strategies: []StrategyEntry{
			{ID: "s1", Exchange: "binance", Strategy: "grid2", Mode: "live",
				Config: []byte(`{"symbol":"BTCUSDT"}`)},
		},
	}

	yamlBytes, err := buildUserYAML(uc, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlBytes)

	if strings.Contains(yamlStr, "PAPER_TRADE") {
		t.Errorf("live mode should NOT have PAPER_TRADE, got:\n%s", yamlStr)
	}
	if strings.Contains(yamlStr, "publicOnly: true") {
		t.Errorf("live mode with credentials should NOT set publicOnly, got:\n%s", yamlStr)
	}
}
