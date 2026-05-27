package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	go ts.cleanup()
	return ts
}

func (ts *WSTicketStore) Close() {
	select {
	case <-ts.done:
	default:
		close(ts.done)
	}
}

func (ts *WSTicketStore) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ts.done:
			return
		case <-ticker.C:
		}
		ts.mu.Lock()
		now := time.Now()
		for k, t := range ts.tickets {
			if now.After(t.expiresAt) {
				delete(ts.tickets, k)
			}
		}
		ts.mu.Unlock()
	}
}

func (ts *WSTicketStore) Issue(userID string) string {
	b := make([]byte, 24)
	rand.Read(b)
	ticket := hex.EncodeToString(b)
	ts.mu.Lock()
	ts.tickets[ticket] = &wsTicket{userID: userID, expiresAt: time.Now().Add(30 * time.Second)}
	ts.mu.Unlock()
	return ticket
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
	ticket := api.wsTickets.Issue(userID)
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

	if api.hub == nil {
		wsWrite(conn, r.Context(), nil, []byte(`{"error":"marketdata not available"}`))
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Subscribe to market data
	marketCh, err := api.hub.SubscribeMarket(ctx)
	if err != nil {
		log.Printf("ws market subscribe error: %v", err)
		return
	}
	defer api.hub.Unsubscribe("market", marketCh)

	// Subscribe to user data if container is running
	var userCh chan json.RawMessage
	containers := api.users.GetByUser(userID)
	for _, uc := range containers {
		if uc.Status != StatusRunning {
			continue
		}
		containerAddr := api.container.ContainerGRPCAddr(userID, uc.Mode)
		sessions := extractSessionNames(uc)
		ch, err := api.hub.SubscribeUserData(ctx, userID, containerAddr, sessions)
		if err != nil {
			log.Printf("ws user data subscribe error for %s (%s): %v", userID, uc.Mode, err)
			continue
		}
		userCh = ch
		defer api.hub.Unsubscribe("user:"+userID, userCh)
		break
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

func extractSessionNames(uc *UserContainer) []string {
	seen := map[string]bool{}
	for _, s := range uc.Strategies {
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
