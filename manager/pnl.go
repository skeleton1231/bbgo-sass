package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

type PnLTrade struct {
	Symbol      string
	Side        string
	Price       float64
	Quantity    float64
	Fee         float64
	FeeCurrency string
	TradedAt    string
}

type SymbolPnL struct {
	Symbol           string  `json:"symbol"`
	RealizedPnL      float64 `json:"realizedPnl"`
	TotalBuys        float64 `json:"totalBuys"`
	TotalSells       float64 `json:"totalSells"`
	BuyVolume        float64 `json:"buyVolume"`
	SellVolume       float64 `json:"sellVolume"`
	TotalFees        float64 `json:"totalFees"`
	TradeCount       int     `json:"tradeCount"`
	WinningTrades    int     `json:"winningTrades"`
	LosingTrades     int     `json:"losingTrades"`
	AvgBuyPrice      float64 `json:"avgBuyPrice"`
	AvgSellPrice     float64 `json:"avgSellPrice"`
	OpenPosition     float64 `json:"openPosition"`
	OpenPositionCost float64 `json:"openPositionCost"`
	UnrealizedPnL    float64 `json:"unrealizedPnl"`
	CurrentPrice     float64 `json:"currentPrice"`
}

type DailyPnl struct {
	Date string  `json:"date"`
	PnL  float64 `json:"pnl"`
}

type PnlCurvePoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type PnLReport struct {
	TotalRealizedPnL   float64         `json:"totalRealizedPnl"`
	TotalUnrealizedPnL float64         `json:"totalUnrealizedPnl"`
	TotalFees          float64         `json:"totalFees"`
	TotalTrades        int             `json:"totalTrades"`
	WinningTrades      int             `json:"winningTrades"`
	LosingTrades       int             `json:"losingTrades"`
	WinRate            float64         `json:"winRate"`
	Symbols            []SymbolPnL     `json:"symbols"`
	DailyBreakdown     []DailyPnl      `json:"dailyBreakdown"`
	PnlCurve           []PnlCurvePoint `json:"pnlCurve"`
}

type fifoQueue struct {
	items []fifoItem
	head  int
}

type fifoItem struct {
	price    float64
	quantity float64
}

func (q *fifoQueue) push(price, quantity float64) {
	q.items = append(q.items, fifoItem{price: price, quantity: quantity})
}

func (q *fifoQueue) pop(quantity float64) (costBasis float64, remaining float64) {
	remaining = quantity
	for remaining > 1e-12 && q.head < len(q.items) {
		item := &q.items[q.head]
		if item.quantity <= remaining {
			costBasis += item.quantity * item.price
			remaining -= item.quantity
			q.head++
		} else {
			costBasis += remaining * item.price
			item.quantity -= remaining
			remaining = 0
		}
	}
	return costBasis, remaining
}

func (q *fifoQueue) remaining() []fifoItem {
	return q.items[q.head:]
}

func isQuoteCurrency(currency string) bool {
	switch currency {
	case "", "USDT", "USDC", "BUSD", "TUSD", "DAI", "FDUSD":
		return true
	}
	return false
}

func feeInQuoteCurrency(fee float64, feeCurrency string, tradePrice float64) float64 {
	fee = math.Abs(fee)
	if isQuoteCurrency(feeCurrency) {
		return fee
	}
	return fee * tradePrice
}

func calculatePnL(trades []BBGoTrade) PnLReport {
	grouped := make(map[string][]PnLTrade)
	for _, t := range trades {
		price, _ := strconv.ParseFloat(string(t.Price), 64)
		qty, _ := strconv.ParseFloat(string(t.Quantity), 64)
		fee, _ := strconv.ParseFloat(string(t.Fee), 64)
		if price == 0 || qty == 0 {
			continue
		}
		grouped[t.Symbol] = append(grouped[t.Symbol], PnLTrade{
			Symbol:      t.Symbol,
			Side:        t.Side,
			Price:       price,
			Quantity:    qty,
			Fee:         fee,
			FeeCurrency: t.FeeCurrency,
			TradedAt:    t.TradedAt,
		})
	}

	report := PnLReport{}
	var symbols []SymbolPnL

	// Track per-day realized P&L across all symbols
	dailyMap := make(map[string]float64)

	for sym, symTrades := range grouped {
		sort.Slice(symTrades, func(i, j int) bool {
			return symTrades[i].TradedAt < symTrades[j].TradedAt
		})

		symPnL := SymbolPnL{Symbol: sym}
		buyQueue := &fifoQueue{}

		for _, t := range symTrades {
			symPnL.TradeCount++
			feeQuote := feeInQuoteCurrency(t.Fee, t.FeeCurrency, t.Price)
			symPnL.TotalFees += feeQuote

			day := ""
			if len(t.TradedAt) >= 10 {
				day = t.TradedAt[:10]
			}

			if t.Side == "BUY" {
				symPnL.TotalBuys += t.Quantity
				symPnL.BuyVolume += t.Price * t.Quantity
				buyQueue.push(t.Price, t.Quantity)
				if day != "" {
					dailyMap[day] -= feeQuote
				}
			} else {
				symPnL.TotalSells += t.Quantity
				symPnL.SellVolume += t.Price * t.Quantity

				costBasis, unmatched := buyQueue.pop(t.Quantity)
				if unmatched > 0 {
					costBasis += unmatched * t.Price
				}
				realized := (t.Price * t.Quantity) - costBasis
				symPnL.RealizedPnL += realized

				if realized > 0 {
					symPnL.WinningTrades++
				} else if realized < 0 {
					symPnL.LosingTrades++
				}

				if day != "" {
					dailyMap[day] += realized - feeQuote
				}
			}
		}

		for _, item := range buyQueue.remaining() {
			symPnL.OpenPosition += item.quantity
			symPnL.OpenPositionCost += item.quantity * item.price
		}

		if symPnL.TotalBuys > 0 {
			symPnL.AvgBuyPrice = symPnL.BuyVolume / symPnL.TotalBuys
		}
		if symPnL.TotalSells > 0 {
			symPnL.AvgSellPrice = symPnL.SellVolume / symPnL.TotalSells
		}

		report.TotalRealizedPnL += symPnL.RealizedPnL
		report.TotalFees += symPnL.TotalFees
		report.TotalTrades += symPnL.TradeCount
		report.WinningTrades += symPnL.WinningTrades
		report.LosingTrades += symPnL.LosingTrades

		symbols = append(symbols, symPnL)
	}

	sort.Slice(symbols, func(i, j int) bool {
		return math.Abs(symbols[i].RealizedPnL) > math.Abs(symbols[j].RealizedPnL)
	})

	if report.WinningTrades+report.LosingTrades > 0 {
		report.WinRate = float64(report.WinningTrades) / float64(report.WinningTrades+report.LosingTrades) * 100
	}

	report.Symbols = symbols

	// Build daily breakdown sorted by date
	days := make([]DailyPnl, 0, len(dailyMap))
	for d, pnl := range dailyMap {
		days = append(days, DailyPnl{Date: d, PnL: math.Round(pnl*100) / 100})
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date < days[j].Date
	})
	report.DailyBreakdown = days

	// Build cumulative P&L curve from daily breakdown
	cum := 0.0
	for _, d := range days {
		cum += d.PnL
		ts := dateToTimestamp(d.Date)
		report.PnlCurve = append(report.PnlCurve, PnlCurvePoint{
			Time:  ts,
			Value: math.Round(cum*100) / 100,
		})
	}

	return report
}

// enrichUnrealizedPnl fills in UnrealizedPnL for each symbol with open positions
// using the provided price lookup function.
func enrichUnrealizedPnl(report *PnLReport, priceLookup func(symbol string) (float64, error)) {
	if priceLookup == nil {
		return
	}
	totalUnrealized := 0.0
	for i := range report.Symbols {
		sym := &report.Symbols[i]
		if sym.OpenPosition <= 0 {
			continue
		}
		price, err := priceLookup(sym.Symbol)
		if err != nil || price <= 0 {
			continue
		}
		sym.CurrentPrice = price
		marketValue := price * sym.OpenPosition
		sym.UnrealizedPnL = marketValue - sym.OpenPositionCost
		totalUnrealized += sym.UnrealizedPnL
	}
	report.TotalUnrealizedPnL = totalUnrealized
}

func dateToTimestamp(date string) int64 {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func fmtFloat(v float64, prec int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return fmt.Sprintf("%.*f", prec, v)
}
