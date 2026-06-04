package main

import (
	"sort"
	"strconv"
)

type positionTag struct {
	tag    string
	netPos float64
}

// computePositionTags processes trades chronologically and computes position action tags.
// Trades must be sorted by tradedAt ASC before calling this function.
// initialNet is the cumulative net position before the first trade.
func computePositionTags(trades []BBGoTrade, initialNet float64) []positionTag {
	tags := make([]positionTag, len(trades))
	net := initialNet

	for i, t := range trades {
		qty := parseFloat(t.Quantity)
		if t.Side == "SELL" {
			qty = -qty
		}
		prev := net
		net += qty

		var tag string
		switch {
		case prev == 0 && net != 0:
			tag = "open"
		case prev != 0 && net == 0:
			tag = "close"
		case prev != 0:
			if t.Side == "BUY" {
				tag = "add"
			} else {
				tag = "reduce"
			}
		default:
			tag = "trade"
		}
		tags[i] = positionTag{tag: tag, netPos: net}
		trades[i].PositionAction = tag
		trades[i].NetPosition = net
	}
	return tags
}

func parseFloat[T ~string](s T) float64 {
	v, _ := strconv.ParseFloat(string(s), 64)
	return v
}

// sortTradesASC sorts trades by tradedAt ascending. For same-timestamp trades
// (grid strategy kline fills), higher array indices are processed first to match
// the bbgo API DESC order convention (reversing recreates gid ASC insertion order).
func sortTradesASC(trades []BBGoTrade) {
	sort.SliceStable(trades, func(i, j int) bool {
		if trades[i].TradedAt < trades[j].TradedAt {
			return true
		}
		if trades[i].TradedAt > trades[j].TradedAt {
			return false
		}
		return j < i
	})
}
