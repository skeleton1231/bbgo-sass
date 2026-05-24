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
}

type wsTicket struct {
	userID    string
	expiresAt time.Time
}

func NewWSTicketStore() *WSTicketStore {
	ts := &WSTicketStore{tickets: make(map[string]*wsTicket)}
	go ts.cleanup()
	return ts
}

func (ts *WSTicketStore) cleanup() {
	for {
		time.Sleep(30 * time.Second)
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
	uc, found := api.users.Get(userID)
	if found && uc.Status == StatusRunning {
		containerAddr := api.container.ContainerGRPCAddr(userID)
		sessions := extractSessionNames(uc)
		userCh, err = api.hub.SubscribeUserData(ctx, userID, containerAddr, sessions)
		if err != nil {
			log.Printf("ws user data subscribe error for %s: %v", userID, err)
		} else {
			defer api.hub.Unsubscribe("user:"+userID, userCh)
		}
	}

	var writeMu sync.Mutex

	// Forward messages to WebSocket
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
				wsWrite(conn, ctx, &writeMu, wsMsg)
			case msg, ok := <-userCh:
				if !ok {
					return
				}
				wsMsg, _ := json.Marshal(WSMessage{Type: "userData", Data: msg})
				wsWrite(conn, ctx, &writeMu, wsMsg)
			}
		}
	}()

	// Read from WebSocket (keep-alive / client commands)
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	conn.Close(websocket.StatusNormalClosure, "")
}

func (cm *ContainerManager) ContainerGRPCAddr(userID string) string {
	return fmt.Sprintf("%s:%d", cm.containerName(userID), cm.cfg.BBGOGRPCPort)
}

func wsWrite(conn *websocket.Conn, ctx context.Context, mu *sync.Mutex, msg []byte) {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	conn.Write(ctx, websocket.MessageText, msg)
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
