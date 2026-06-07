package main

import (
	"testing"
)

func TestComputePositionTags_OpenClose(t *testing.T) {
	trades := []BBGoTrade{
		{Side: "BUY", Quantity: "1", TradedAt: "2024-01-01T00:00:00Z"},
		{Side: "SELL", Quantity: "1", TradedAt: "2024-01-02T00:00:00Z"},
	}
	tags := computePositionTags(trades, 0)
	if tags[0].tag != "open" {
		t.Errorf("trade 0: want open, got %q", tags[0].tag)
	}
	if tags[0].netPos != 1 {
		t.Errorf("trade 0 net: want 1, got %f", tags[0].netPos)
	}
	if tags[1].tag != "close" {
		t.Errorf("trade 1: want close, got %q", tags[1].tag)
	}
	if tags[1].netPos != 0 {
		t.Errorf("trade 1 net: want 0, got %f", tags[1].netPos)
	}
}

func TestComputePositionTags_AddReduce(t *testing.T) {
	trades := []BBGoTrade{
		{Side: "BUY", Quantity: "1", TradedAt: "2024-01-01T00:00:00Z"},
		{Side: "BUY", Quantity: "1", TradedAt: "2024-01-02T00:00:00Z"},
		{Side: "SELL", Quantity: "1", TradedAt: "2024-01-03T00:00:00Z"},
	}
	tags := computePositionTags(trades, 0)
	if tags[0].tag != "open" {
		t.Errorf("trade 0: want open, got %q", tags[0].tag)
	}
	if tags[1].tag != "add" {
		t.Errorf("trade 1: want add, got %q", tags[1].tag)
	}
	if tags[2].tag != "reduce" {
		t.Errorf("trade 2: want reduce, got %q", tags[2].tag)
	}
	if tags[2].netPos != 1 {
		t.Errorf("trade 2 net: want 1, got %f", tags[2].netPos)
	}
}

func TestComputePositionTags_WithInitialNet(t *testing.T) {
	trades := []BBGoTrade{
		{Side: "SELL", Quantity: "1", TradedAt: "2024-01-01T00:00:00Z"},
	}
	tags := computePositionTags(trades, 1)
	if tags[0].tag != "close" {
		t.Errorf("with initial=1: want close, got %q", tags[0].tag)
	}
	if tags[0].netPos != 0 {
		t.Errorf("net: want 0, got %f", tags[0].netPos)
	}
}

func TestComputePositionTags_TradeFromZero(t *testing.T) {
	trades := []BBGoTrade{
		{Side: "BUY", Quantity: "1", TradedAt: "2024-01-01T00:00:00Z"},
		{Side: "BUY", Quantity: "1", TradedAt: "2024-01-02T00:00:00Z"},
	}
	tags := computePositionTags(trades, 0)
	if tags[0].tag != "open" {
		t.Errorf("want open, got %q", tags[0].tag)
	}
	if tags[1].tag != "add" {
		t.Errorf("want add, got %q", tags[1].tag)
	}
}

func TestParseFloat(t *testing.T) {
	if v := parseFloat("3.14"); v != 3.14 {
		t.Errorf("parseFloat(3.14) = %f", v)
	}
	if v := parseFloat(""); v != 0 {
		t.Errorf("parseFloat('') = %f, want 0", v)
	}
	if v := parseFloat("abc"); v != 0 {
		t.Errorf("parseFloat('abc') = %f, want 0", v)
	}
}

func TestSortTradesASC(t *testing.T) {
	trades := []BBGoTrade{
		{TradedAt: "2024-01-03T00:00:00Z", GID: 3},
		{TradedAt: "2024-01-01T00:00:00Z", GID: 1},
		{TradedAt: "2024-01-02T00:00:00Z", GID: 2},
	}
	sortTradesASC(trades)
	if trades[0].TradedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("not sorted ASC: [0]=%s", trades[0].TradedAt)
	}
	if trades[2].TradedAt != "2024-01-03T00:00:00Z" {
		t.Errorf("not sorted ASC: [2]=%s", trades[2].TradedAt)
	}
}

func TestSortTradesASC_SameTimestamp(t *testing.T) {
	trades := []BBGoTrade{
		{TradedAt: "2024-01-01T00:00:00Z", GID: 1},
		{TradedAt: "2024-01-01T00:00:00Z", GID: 2},
		{TradedAt: "2024-01-01T00:00:00Z", GID: 3},
	}
	sortTradesASC(trades)
	if trades[0].GID != 3 {
		t.Errorf("same-timestamp sort: [0].GID = %d, want 3", trades[0].GID)
	}
	if trades[2].GID != 1 {
		t.Errorf("same-timestamp sort: [2].GID = %d, want 1", trades[2].GID)
	}
}

func TestComputePositionTags_WritesBackToTrades(t *testing.T) {
	trades := []BBGoTrade{
		{Side: "BUY", Quantity: "1", TradedAt: "2024-01-01T00:00:00Z"},
	}
	computePositionTags(trades, 0)
	if trades[0].PositionAction != "open" {
		t.Errorf("PositionAction = %q, want open", trades[0].PositionAction)
	}
	if trades[0].NetPosition != 1 {
		t.Errorf("NetPosition = %f, want 1", trades[0].NetPosition)
	}
}
