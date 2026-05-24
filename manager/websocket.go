package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

type WSMessage struct {
	Type string          `json:"type"` // "market" | "userData"
	Data json.RawMessage `json:"data"`
}

func (api *API) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		userID = r.URL.Query().Get("userId")
		if !isValidUUID(userID) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
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
		userCh, err = api.hub.SubscribeUserData(ctx, userID, containerAddr)
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
