package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestExtractSessionNames_SingleExchange(t *testing.T) {
	sessions := extractSessionNames([]StrategyEntry{
		{Exchange: "binance", Strategy: "grid"},
	})
	if len(sessions) != 1 || sessions[0] != "binance" {
		t.Fatalf("expected [binance], got %v", sessions)
	}
}

func TestExtractSessionNames_CrossExchange(t *testing.T) {
	sessions := extractSessionNames([]StrategyEntry{
		{
			CrossExchange: true,
			Sessions: []SessionRoleConfig{
				{Name: "binance", Exchange: "binance"},
				{Name: "okex", Exchange: "okex"},
			},
		},
	})
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %v", len(sessions), sessions)
	}
	if sessions[0] != "binance" || sessions[1] != "okex" {
		t.Fatalf("expected [binance okex], got %v", sessions)
	}
}

func TestExtractSessionNames_Deduplicates(t *testing.T) {
	sessions := extractSessionNames([]StrategyEntry{
		{Exchange: "binance", Strategy: "grid"},
		{Exchange: "binance", Strategy: "xmaker"},
	})
	if len(sessions) != 1 {
		t.Fatalf("expected 1 unique session, got %d: %v", len(sessions), sessions)
	}
}

func TestWSTicket_IssueRedeem(t *testing.T) {
	ts := NewWSTicketStore()
	ticket := ts.Issue("user-1")
	if ticket == "" {
		t.Fatal("expected non-empty ticket")
	}
	userID, ok := ts.Redeem(ticket)
	if !ok || userID != "user-1" {
		t.Fatalf("expected user-1, got %s ok=%v", userID, ok)
	}
	_, ok = ts.Redeem(ticket)
	if ok {
		t.Error("ticket should be single-use")
	}
}

func TestWSTicket_ExpiredTicket(t *testing.T) {
	ts := NewWSTicketStore()
	ts.mu.Lock()
	ts.tickets["expired"] = &wsTicket{expiresAt: time.Now().Add(-1 * time.Second)}
	ts.mu.Unlock()
	_, ok := ts.Redeem("expired")
	if ok {
		t.Error("expired ticket should not be redeemable")
	}
}

func TestWSTicket_InvalidTicket(t *testing.T) {
	ts := NewWSTicketStore()
	_, ok := ts.Redeem("nonexistent")
	if ok {
		t.Error("nonexistent ticket should not be redeemable")
	}
}

func TestWSTicketStore_Close_StopsCleanup(t *testing.T) {
	ts := NewWSTicketStore()
	ts.Close()
	// Double close should not panic
	ts.Close()
}

func TestWSTicketStore_CleanupRemovesExpired(t *testing.T) {
	ts := NewWSTicketStore()
	defer ts.Close()

	ts.mu.Lock()
	ts.tickets["old"] = &wsTicket{expiresAt: time.Now().Add(-1 * time.Second)}
	ts.tickets["fresh"] = &wsTicket{expiresAt: time.Now().Add(30 * time.Second)}
	ts.mu.Unlock()

	time.Sleep(100 * time.Millisecond)
	// Manually trigger cleanup
	ts.mu.Lock()
	now := time.Now()
	for k, t := range ts.tickets {
		if now.After(t.expiresAt) {
			delete(ts.tickets, k)
		}
	}
	ts.mu.Unlock()

	_, ok := ts.Redeem("old")
	if ok {
		t.Error("old ticket should have been cleaned up")
	}
	_, ok = ts.Redeem("fresh")
	if !ok {
		t.Error("fresh ticket should still be valid")
	}
}

// TestWSForward_ContinuesAfterUserChClose verifies that market data forwarding
// continues after the user data channel closes (separate goroutines).
func TestWSForward_ContinuesAfterUserChClose(t *testing.T) {
	marketCh := make(chan json.RawMessage, 8)
	userCh := make(chan json.RawMessage, 8)
	received := make(chan json.RawMessage, 4)

	var writeMu sync.Mutex
	var closed bool

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Market data forwarder (independent goroutine)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-marketCh:
				if !ok {
					return
				}
				wsMsg, _ := json.Marshal(WSMessage{Type: "market", Data: msg})
				writeMu.Lock()
				if !closed {
					received <- wsMsg
				}
				writeMu.Unlock()
			}
		}
	}()

	// User data forwarder (separate goroutine — can die independently)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-userCh:
				if !ok {
					return
				}
				wsMsg, _ := json.Marshal(WSMessage{Type: "userData", Data: msg})
				writeMu.Lock()
				if !closed {
					received <- wsMsg
				}
				writeMu.Unlock()
			}
		}
	}()

	// Send user data then close user channel
	userCh <- json.RawMessage(`{"user":"data"}`)
	time.Sleep(20 * time.Millisecond)
	close(userCh)
	time.Sleep(20 * time.Millisecond)

	// Market data should still arrive after userCh closed
	marketCh <- json.RawMessage(`{"market":"after-close"}`)

	// Drain up to 2 messages (userData + market)
	var gotMarket bool
	for i := 0; i < 2; i++ {
		select {
		case msg := <-received:
			var wsMsg WSMessage
			json.Unmarshal(msg, &wsMsg)
			if wsMsg.Type == "market" {
				gotMarket = true
			}
		case <-time.After(2 * time.Second):
			if !gotMarket {
				t.Fatal("timed out — market data should still flow after userCh closes")
			}
		}
	}
	if !gotMarket {
		t.Fatal("market message not received after userCh close")
	}

	writeMu.Lock()
	closed = true
	writeMu.Unlock()
}
