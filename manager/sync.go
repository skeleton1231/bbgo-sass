package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type Syncer struct {
	users     *UserContainerManager
	cfg       *Config
	container *ContainerManager
	client    *http.Client
}

func NewSyncer(users *UserContainerManager, cfg *Config, cm *ContainerManager) *Syncer {
	return &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *Syncer) LoadUsersFromSupabase() ([]*UserContainer, error) {
	url := s.cfg.SupabaseURL + "/rest/v1/user_containers?select=*"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", s.cfg.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var raw []struct {
		UserID     string          `json:"user_id"`
		Status     string          `json:"status"`
		Strategies json.RawMessage `json:"strategies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	users := make([]*UserContainer, len(raw))
	for i, r := range raw {
		uc := &UserContainer{
			UserID: r.UserID,
			Status: r.Status,
		}
		if len(r.Strategies) > 0 {
			json.Unmarshal(r.Strategies, &uc.Strategies)
		}
		users[i] = uc
	}
	return users, nil
}

func (s *Syncer) UpsertUser(uc *UserContainer) {
	payload, _ := json.Marshal(map[string]interface{}{
		"user_id":    uc.UserID,
		"status":     uc.Status,
		"strategies": uc.Strategies,
	})

	url := s.cfg.SupabaseURL + "/rest/v1/user_containers"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("apikey", s.cfg.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicate")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("upsert user %s failed: %v", uc.UserID, err)
		return
	}
	resp.Body.Close()
}

func (s *Syncer) SyncUser(userID string) {
	uc, ok := s.users.Get(userID)
	if !ok {
		return
	}
	s.UpsertUser(uc)
	s.syncUserData(uc)
}

func (s *Syncer) SyncAll() {
	users := s.users.ListUsers()
	for _, uc := range users {
		s.UpsertUser(uc)
		if uc.Status == StatusRunning {
			s.syncUserData(uc)
		}
	}
}

func (s *Syncer) syncUserData(uc *UserContainer) {
	dir := s.container.userDir(uc.UserID)
	dbPath := dir + "/bbgo.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return
	}
	defer db.Close()

	s.syncOrders(uc.UserID, db)
	s.syncTrades(uc.UserID, db)
}

func (s *Syncer) syncOrders(userID string, db *sql.DB) {
	rows, err := db.Query("SELECT exchange, order_id, symbol, side, price, quantity, executed_quantity, status, order_type, created_at FROM orders ORDER BY created_at DESC LIMIT 200")
	if err != nil {
		log.Printf("sync orders query failed for user %s: %v", userID, err)
		return
	}
	defer rows.Close()

	var orders []map[string]interface{}
	for rows.Next() {
		var exchange, symbol, side, status, orderType, createdAt string
		var orderID, price, quantity, executedQuantity string
		rows.Scan(&exchange, &orderID, &symbol, &side, &price, &quantity, &executedQuantity, &status, &orderType, &createdAt)
		orders = append(orders, map[string]interface{}{
			"user_id":           userID,
			"exchange":          exchange,
			"order_id":          orderID,
			"symbol":            symbol,
			"side":              side,
			"price":             price,
			"quantity":          quantity,
			"executed_quantity": executedQuantity,
			"status":            status,
			"order_type":        orderType,
			"created_at":        createdAt,
		})
	}

	if len(orders) == 0 {
		return
	}

	payload, _ := json.Marshal(orders)
	url := s.cfg.SupabaseURL + "/rest/v1/sync_orders"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("apikey", s.cfg.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicate")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("sync orders for user %s failed: %v", userID, err)
		return
	}
	resp.Body.Close()
	log.Printf("synced %d orders for user %s", len(orders), userID)
}

func (s *Syncer) syncTrades(userID string, db *sql.DB) {
	rows, err := db.Query("SELECT id, exchange, symbol, side, price, quantity, quote_quantity, fee, fee_currency, is_buyer, traded_at FROM trades ORDER BY traded_at DESC LIMIT 200")
	if err != nil {
		log.Printf("sync trades query failed for user %s: %v", userID, err)
		return
	}
	defer rows.Close()

	var trades []map[string]interface{}
	for rows.Next() {
		var id, exchange, symbol, side, price, quantity, quoteQuantity, fee, feeCurrency, isBuyer, tradedAt string
		rows.Scan(&id, &exchange, &symbol, &side, &price, &quantity, &quoteQuantity, &fee, &feeCurrency, &isBuyer, &tradedAt)
		trades = append(trades, map[string]interface{}{
			"user_id":        userID,
			"trade_id":       id,
			"exchange":       exchange,
			"symbol":         symbol,
			"side":           side,
			"price":          price,
			"quantity":       quantity,
			"quote_quantity": quoteQuantity,
			"fee":            fee,
			"fee_currency":   feeCurrency,
			"is_buyer":       isBuyer,
			"traded_at":      tradedAt,
		})
	}

	if len(trades) == 0 {
		return
	}

	payload, _ := json.Marshal(trades)
	url := s.cfg.SupabaseURL + "/rest/v1/sync_trades"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("apikey", s.cfg.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicate")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("sync trades for user %s failed: %v", userID, err)
		return
	}
	resp.Body.Close()
	log.Printf("synced %d trades for user %s", len(trades), userID)
}
