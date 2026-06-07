package main

import (
	"math"
	"testing"
)

func TestFifoQueue_PushPop(t *testing.T) {
	q := &fifoQueue{}
	q.push(100, 10)
	q.push(105, 5)

	cost, rem := q.pop(8)
	if rem != 0 {
		t.Fatalf("remaining should be 0, got %f", rem)
	}
	if cost != 800 {
		t.Fatalf("cost should be 800, got %f", cost)
	}
}

func TestFifoQueue_PopAcrossItems(t *testing.T) {
	q := &fifoQueue{}
	q.push(100, 10)
	q.push(105, 5)

	cost, rem := q.pop(12)
	if rem != 0 {
		t.Fatalf("remaining should be 0, got %f", rem)
	}
	if cost != 1210 {
		t.Fatalf("cost should be 1210, got %f", cost)
	}
}

func TestFifoQueue_PopMoreThanAvailable(t *testing.T) {
	q := &fifoQueue{}
	q.push(100, 5)

	cost, rem := q.pop(10)
	if cost != 500 {
		t.Fatalf("cost should be 500, got %f", cost)
	}
	if rem != 5 {
		t.Fatalf("remaining should be 5, got %f", rem)
	}
}

func TestFifoQueue_PopEmpty(t *testing.T) {
	q := &fifoQueue{}
	cost, rem := q.pop(10)
	if cost != 0 {
		t.Fatalf("cost should be 0, got %f", cost)
	}
	if rem != 10 {
		t.Fatalf("remaining should be 10, got %f", rem)
	}
}

func TestFifoQueue_Remaining(t *testing.T) {
	q := &fifoQueue{}
	q.push(100, 10)
	q.push(105, 5)
	q.pop(8)

	items := q.remaining()
	if len(items) != 2 {
		t.Fatalf("expected 2 remaining items, got %d", len(items))
	}
	if items[0].quantity != 2 || items[0].price != 100 {
		t.Fatalf("first remaining: want qty=2 price=100, got qty=%f price=%f", items[0].quantity, items[0].price)
	}
	if items[1].quantity != 5 || items[1].price != 105 {
		t.Fatalf("second remaining: want qty=5 price=105, got qty=%f price=%f", items[1].quantity, items[1].price)
	}
}

func TestIsQuoteCurrency(t *testing.T) {
	tests := []struct {
		currency string
		want     bool
	}{
		{"USDT", true}, {"USDC", true}, {"BUSD", true},
		{"TUSD", true}, {"DAI", true}, {"FDUSD", true}, {"", true},
		{"BTC", false}, {"ETH", false}, {"BNB", false},
	}
	for _, tt := range tests {
		if got := isQuoteCurrency(tt.currency); got != tt.want {
			t.Errorf("isQuoteCurrency(%q) = %v, want %v", tt.currency, got, tt.want)
		}
	}
}

func TestFeeInQuoteCurrency(t *testing.T) {
	if got := feeInQuoteCurrency(0.1, "USDT", 50000); got != 0.1 {
		t.Errorf("fee in quote: want 0.1, got %f", got)
	}
	if got := feeInQuoteCurrency(0.001, "BTC", 50000); got != 50 {
		t.Errorf("fee converted: want 50, got %f", got)
	}
	if got := feeInQuoteCurrency(-0.1, "USDT", 50000); got != 0.1 {
		t.Errorf("negative fee abs: want 0.1, got %f", got)
	}
}

func TestCalculatePnL_BuySellCycle(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0.001", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0.001", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
	}
	report := calculatePnL(trades)
	if len(report.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(report.Symbols))
	}
	sym := report.Symbols[0]
	if sym.RealizedPnL != 5000 {
		t.Errorf("realized pnl = %f, want 5000", sym.RealizedPnL)
	}
	if sym.OpenPosition != 0 {
		t.Errorf("open position = %f, want 0", sym.OpenPosition)
	}
	if sym.WinningTrades != 1 {
		t.Errorf("winning trades = %d, want 1", sym.WinningTrades)
	}
}

func TestCalculatePnL_PartialClose(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "2", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
	}
	report := calculatePnL(trades)
	sym := report.Symbols[0]
	if sym.RealizedPnL != 5000 {
		t.Errorf("realized = %f, want 5000", sym.RealizedPnL)
	}
	if sym.OpenPosition != 1 {
		t.Errorf("open position = %f, want 1", sym.OpenPosition)
	}
	if sym.OpenPositionCost != 50000 {
		t.Errorf("open position cost = %f, want 50000", sym.OpenPositionCost)
	}
}

func TestCalculatePnL_LosingTrade(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "45000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
	}
	report := calculatePnL(trades)
	if report.TotalRealizedPnL != -5000 {
		t.Errorf("total realized = %f, want -5000", report.TotalRealizedPnL)
	}
	if report.LosingTrades != 1 {
		t.Errorf("losing trades = %d, want 1", report.LosingTrades)
	}
}

func TestCalculatePnL_MultipleSymbols(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "ETHUSDT", Side: "BUY", Price: "3000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "52000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
		{Symbol: "ETHUSDT", Side: "SELL", Price: "2800", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
	}
	report := calculatePnL(trades)
	if len(report.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(report.Symbols))
	}
	// BTC: 52000-50000=2000, ETH: 2800-3000=-200, total=1800
	if report.TotalRealizedPnL != 1800 {
		t.Errorf("total realized = %f, want 1800", report.TotalRealizedPnL)
	}
}

func TestCalculatePnL_Empty(t *testing.T) {
	report := calculatePnL(nil)
	if report.TotalRealizedPnL != 0 || len(report.Symbols) != 0 {
		t.Errorf("empty input should produce zero report")
	}
}

func TestCalculatePnL_ZeroPriceSkipped(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "0", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
	}
	report := calculatePnL(trades)
	if len(report.Symbols) != 0 {
		t.Errorf("zero-price trades should be skipped")
	}
}

func TestCalculatePnL_DailyBreakdown(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T10:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T10:00:00Z"},
	}
	report := calculatePnL(trades)
	if len(report.DailyBreakdown) != 2 {
		t.Fatalf("expected 2 daily entries, got %d", len(report.DailyBreakdown))
	}
	if report.DailyBreakdown[0].Date != "2024-01-01" {
		t.Errorf("day 1 date = %q", report.DailyBreakdown[0].Date)
	}
	if report.DailyBreakdown[1].PnL != 5000 {
		t.Errorf("day 2 pnl = %f, want 5000", report.DailyBreakdown[1].PnL)
	}
}

func TestCalculatePnL_PnlCurve(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
	}
	report := calculatePnL(trades)
	if len(report.PnlCurve) != 2 {
		t.Fatalf("expected 2 curve points, got %d", len(report.PnlCurve))
	}
	if report.PnlCurve[1].Value != 5000 {
		t.Errorf("curve[1] value = %f, want 5000", report.PnlCurve[1].Value)
	}
}

func TestCalculatePnL_WinRate(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "54000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-03T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "53000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-04T00:00:00Z"},
	}
	report := calculatePnL(trades)
	if report.WinRate != 50.0 {
		t.Errorf("win rate = %f, want 50", report.WinRate)
	}
}

func TestCalculatePnL_FIFOOrdering(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "40000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "48000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-03T00:00:00Z"},
	}
	report := calculatePnL(trades)
	sym := report.Symbols[0]
	if sym.RealizedPnL != 8000 {
		t.Errorf("realized = %f, want 8000 (FIFO matches 40000)", sym.RealizedPnL)
	}
	if sym.OpenPosition != 1 || sym.OpenPositionCost != 50000 {
		t.Errorf("open pos=%f cost=%f, want 1/50000", sym.OpenPosition, sym.OpenPositionCost)
	}
}

func TestCalculatePnL_AvgPrices(t *testing.T) {
	trades := []BBGoTrade{
		{Symbol: "BTCUSDT", Side: "BUY", Price: "50000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-01T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "BUY", Price: "60000", Quantity: "1", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-02T00:00:00Z"},
		{Symbol: "BTCUSDT", Side: "SELL", Price: "55000", Quantity: "2", Fee: "0", FeeCurrency: "USDT", TradedAt: "2024-01-03T00:00:00Z"},
	}
	report := calculatePnL(trades)
	sym := report.Symbols[0]
	if sym.AvgBuyPrice != 55000 {
		t.Errorf("avg buy = %f, want 55000", sym.AvgBuyPrice)
	}
	if sym.AvgSellPrice != 55000 {
		t.Errorf("avg sell = %f, want 55000", sym.AvgSellPrice)
	}
}

func TestEnrichUnrealizedPnl(t *testing.T) {
	report := &PnLReport{
		Symbols: []SymbolPnL{
			{Symbol: "BTCUSDT", OpenPosition: 1, OpenPositionCost: 50000},
			{Symbol: "ETHUSDT", OpenPosition: 0},
		},
	}
	lookup := func(symbol string) (float64, error) {
		if symbol == "BTCUSDT" {
			return 55000, nil
		}
		return 0, nil
	}
	enrichUnrealizedPnl(report, lookup)
	if report.TotalUnrealizedPnL != 5000 {
		t.Errorf("total unrealized = %f, want 5000", report.TotalUnrealizedPnL)
	}
	if report.Symbols[0].UnrealizedPnL != 5000 {
		t.Errorf("BTC unrealized = %f, want 5000", report.Symbols[0].UnrealizedPnL)
	}
}

func TestEnrichUnrealizedPnl_NilLookup(t *testing.T) {
	report := &PnLReport{Symbols: []SymbolPnL{{Symbol: "BTCUSDT", OpenPosition: 1}}}
	enrichUnrealizedPnl(report, nil)
	if report.TotalUnrealizedPnL != 0 {
		t.Errorf("nil lookup should not modify report")
	}
}

func TestEnrichUnrealizedPnl_ZeroPrice(t *testing.T) {
	report := &PnLReport{
		Symbols: []SymbolPnL{{Symbol: "BADUSDT", OpenPosition: 1, OpenPositionCost: 100}},
	}
	lookup := func(symbol string) (float64, error) { return 0, nil }
	enrichUnrealizedPnl(report, lookup)
	if report.TotalUnrealizedPnL != 0 {
		t.Errorf("zero price should not add unrealized")
	}
}

func TestDateToTimestamp(t *testing.T) {
	ts := dateToTimestamp("2024-01-01")
	if ts == 0 {
		t.Fatal("expected non-zero timestamp")
	}
	ts2 := dateToTimestamp("not-a-date")
	if ts2 != 0 {
		t.Errorf("invalid date should return 0, got %d", ts2)
	}
}

func TestFmtFloat(t *testing.T) {
	if got := fmtFloat(123.456, 2); got != "123.46" {
		t.Errorf("fmtFloat(123.456, 2) = %q, want 123.46", got)
	}
	if got := fmtFloat(math.NaN(), 2); got != "0" {
		t.Errorf("fmtFloat(NaN, 2) = %q, want 0", got)
	}
	if got := fmtFloat(math.Inf(1), 2); got != "0" {
		t.Errorf("fmtFloat(+Inf, 2) = %q, want 0", got)
	}
}
