package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type WSMessage struct {
	Type string          `json:"type"` // "market" | "userData"
	Data json.RawMessage `json:"data"`
}

// WSTicketStore issues short-lived, single-use tickets for WebSocket auth.
// This avoids exposing the manager token in WebSocket URLs.
type WSTicketStore struct {
	mu      sync.Mutex
	tickets map[string]*wsTicket
	done    chan struct{}
}

type wsTicket struct {
	userID    string
	expiresAt time.Time
}

func NewWSTicketStore() *WSTicketStore {
	ts := &WSTicketStore{
		tickets: make(map[string]*wsTicket),
		done:    make(chan struct{}),
	}
	go ts.startCleanup()
	return ts
}

func (ts *WSTicketStore) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ts.done:
			return
		case <-ticker.C:
			ts.purgeExpired()
		}
	}
}

func (ts *WSTicketStore) Close() {
	select {
	case <-ts.done:
	default:
		close(ts.done)
	}
}

func (ts *WSTicketStore) purgeExpired() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	now := time.Now()
	for k, t := range ts.tickets {
		if now.After(t.expiresAt) {
			delete(ts.tickets, k)
		}
	}
}

func (ts *WSTicketStore) Issue(userID string) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("ws ticket entropy: %w", err)
	}
	ticket := hex.EncodeToString(b)
	ts.mu.Lock()
	ts.tickets[ticket] = &wsTicket{userID: userID, expiresAt: time.Now().Add(30 * time.Second)}
	ts.mu.Unlock()
	return ticket, nil
}

func (ts *WSTicketStore) Redeem(ticket string) (string, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t, ok := ts.tickets[ticket]
	if !ok || time.Now().After(t.expiresAt) {
		return "", false
	}
	delete(ts.tickets, ticket)
	return t.userID, true
}

func (api *API) IssueWSTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	ticket, err := api.wsTickets.Issue(userID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "failed to issue ticket")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ticket": ticket})
}

func (api *API) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "missing ticket", http.StatusUnauthorized)
		return
	}
	userID, ok := api.wsTickets.Redeem(ticket)
	if !ok {
		http.Error(w, "invalid or expired ticket", http.StatusUnauthorized)
		return
	}

	origins := api.cfg.WSAllowedOrigins
	if len(origins) == 0 {
		origins = []string{"localhost:*", "127.0.0.1:*"}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: origins,
	})
	if err != nil {
		log.Printf("ws accept error: %v", err)
		return
	}
	defer conn.CloseNow()

	// Determine mode for hub selection
	wsMode := r.URL.Query().Get("mode")
	hub := api.hubForMode(wsMode)

	if hub == nil {
		wsWrite(conn, r.Context(), nil, []byte(`{"error":"marketdata not available"}`))
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Subscribe to market data from the mode-appropriate hub
	marketCh, err := hub.SubscribeMarket(ctx)
	if err != nil {
		log.Printf("ws market subscribe error: %v", err)
		return
	}
	defer hub.Unsubscribe("market", marketCh)

	// Subscribe to user data from the user's bbgo container (not the marketdata hub).
	// The hub only manages the subscription channel; the actual gRPC connection dials
	// the container directly via containerAddr, so the hub used here doesn't matter.
	var userCh chan json.RawMessage
	modes := []string{ModeLive, ModePaper}
	if wsMode != "" {
		modes = []string{wsMode}
	}
	for _, mode := range modes {
		instances, _ := api.store.ListInstances(userID, mode)
		for _, inst := range instances {
			if !api.isInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
				continue
			}
			containerAddr := api.container.InstanceGRPCAddr(inst.UserID, inst.Mode, inst.InstanceID)
			sessions := extractSessionNames([]StrategyEntry{{
				Strategy: inst.Strategy, Exchange: inst.Exchange,
				CrossExchange: inst.CrossExchange, Sessions: inst.Sessions,
			}})
			ch, err := hub.SubscribeUserData(ctx, userID, containerAddr, sessions)
			if err != nil {
				log.Printf("ws user data subscribe error for %s/%s: %v", inst.InstanceID, mode, err)
				continue
			}
			userCh = ch
			defer hub.Unsubscribe("user:"+userID, userCh)
			break
		}
		if userCh != nil {
			break
		}
	}

	var writeMu sync.Mutex

	// Forward market data — independent goroutine so user stream failures
	// don't kill market data delivery.
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
				if !wsWrite(conn, ctx, &writeMu, wsMsg) {
					return
				}
			}
		}
	}()

	// Forward user data — separate goroutine; can exit independently
	if userCh != nil {
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
					if !wsWrite(conn, ctx, &writeMu, wsMsg) {
						return
					}
				}
			}
		}()
	}

	// Read from WebSocket (keep-alive / client commands)
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	conn.Close(websocket.StatusNormalClosure, "")
}

func wsWrite(conn *websocket.Conn, ctx context.Context, mu *sync.Mutex, msg []byte) bool {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	err := conn.Write(ctx, websocket.MessageText, msg)
	return err == nil
}

func extractSessionNames(strategies []StrategyEntry) []string {
	seen := map[string]bool{}
	for _, s := range strategies {
		if s.CrossExchange {
			for _, sr := range s.Sessions {
				if !seen[sr.Name] {
					seen[sr.Name] = true
				}
			}
		} else if s.Exchange != "" {
			seen[s.Exchange] = true
		}
	}
	sessions := make([]string, 0, len(seen))
	for s := range seen {
		sessions = append(sessions, s)
	}
	sort.Strings(sessions)
	return sessions
}
