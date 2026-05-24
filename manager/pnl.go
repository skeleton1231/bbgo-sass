package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

type PnLTrade struct {
	Symbol   string
	Side     string
	Price    float64
	Quantity float64
	Fee      float64
	TradedAt string
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
}

type PnLReport struct {
	TotalRealizedPnL float64    `json:"totalRealizedPnl"`
	TotalFees        float64    `json:"totalFees"`
	TotalTrades      int        `json:"totalTrades"`
	WinningTrades    int        `json:"winningTrades"`
	LosingTrades     int        `json:"losingTrades"`
	WinRate          float64    `json:"winRate"`
	Symbols          []SymbolPnL `json:"symbols"`
}

type fifoQueue struct {
	items []fifoItem
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
	for remaining > 1e-12 && len(q.items) > 0 {
		head := q.items[0]
		if head.quantity <= remaining {
			costBasis += head.quantity * head.price
			remaining -= head.quantity
			q.items = q.items[1:]
		} else {
			costBasis += remaining * head.price
			head.quantity -= remaining
			q.items[0] = head
			remaining = 0
		}
	}
	return costBasis, remaining
}

func calculatePnL(trades []BBGoTrade) PnLReport {
	grouped := make(map[string][]PnLTrade)
	for _, t := range trades {
		price, _ := strconv.ParseFloat(t.Price, 64)
		qty, _ := strconv.ParseFloat(t.Quantity, 64)
		fee, _ := strconv.ParseFloat(t.Fee, 64)
		if price == 0 || qty == 0 {
			continue
		}
		grouped[t.Symbol] = append(grouped[t.Symbol], PnLTrade{
			Symbol:   t.Symbol,
			Side:     t.Side,
			Price:    price,
			Quantity: qty,
			Fee:      fee,
			TradedAt: t.TradedAt,
		})
	}

	report := PnLReport{}
	var symbols []SymbolPnL

	for sym, symTrades := range grouped {
		sort.Slice(symTrades, func(i, j int) bool {
			return symTrades[i].TradedAt < symTrades[j].TradedAt
		})

		symPnL := SymbolPnL{Symbol: sym}
		buyQueue := &fifoQueue{}

		for _, t := range symTrades {
			symPnL.TradeCount++
			symPnL.TotalFees += math.Abs(t.Fee)

			if t.Side == "BUY" {
				symPnL.TotalBuys += t.Quantity
				symPnL.BuyVolume += t.Price * t.Quantity
				buyQueue.push(t.Price, t.Quantity)
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
			}
		}

		for _, item := range buyQueue.items {
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
	return report
}

func fmtFloat(v float64, prec int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return fmt.Sprintf("%.*f", prec, v)
}
