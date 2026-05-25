package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Syncer struct {
	users           *UserContainerManager
	cfg             *Config
	container       *ContainerManager
	creds           *CredentialStore
	client          *http.Client
	newBBGoClientFn func(baseURL string) *BBGoClient
}

func NewSyncer(users *UserContainerManager, cfg *Config, cm *ContainerManager) *Syncer {
	return &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func NewSyncerWithCreds(users *UserContainerManager, cfg *Config, cm *ContainerManager, creds *CredentialStore) *Syncer {
	s := NewSyncer(users, cfg, cm)
	s.creds = creds
	return s
}

func (s *Syncer) bbgoClient(userID string) *BBGoClient {
	if s.newBBGoClientFn != nil {
		return s.newBBGoClientFn(s.container.APIURL(userID))
	}
	return NewBBGoClient(s.container.APIURL(userID))
}

func (s *Syncer) supabaseRequest(method, path string, body []byte) (*http.Response, error) {
	url := s.cfg.SupabaseURL + "/rest/v1/" + path
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", s.cfg.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicates")
	return s.client.Do(req)
}

func (s *Syncer) LoadUsersFromSupabase() ([]*UserContainer, error) {
	resp, err := s.supabaseRequest("GET", "user_containers?select=*", nil)
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&raw); err != nil {
		return nil, err
	}

	users := make([]*UserContainer, len(raw))
	for i, r := range raw {
		uc := &UserContainer{
			UserID: r.UserID,
			Status: r.Status,
		}
		if len(r.Strategies) > 0 {
			if err := json.Unmarshal(r.Strategies, &uc.Strategies); err != nil {
				log.Printf("unmarshal strategies for user %s: %v", r.UserID, err)
			}
		}
		users[i] = uc
	}
	return users, nil
}

func (s *Syncer) UpsertUser(uc *UserContainer) {
	payload, err := json.Marshal(map[string]interface{}{
		"user_id":    uc.UserID,
		"status":     uc.Status,
		"strategies": uc.Strategies,
	})
	if err != nil {
		log.Printf("marshal user %s: %v", uc.UserID, err)
		return
	}

	resp, err := s.supabaseRequest("POST", "user_containers?on_conflict=user_id", payload)
	if err != nil {
		log.Printf("upsert user %s failed: %v", uc.UserID, err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("upsert user %s rejected (status %d)", uc.UserID, resp.StatusCode)
	}
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

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // max 5 concurrent syncs
	for _, uc := range users {
		s.UpsertUser(uc)
		if uc.Status != StatusRunning {
			continue
		}
		wg.Add(1)
		go func(uc *UserContainer) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s.syncUserData(uc)
		}(uc)
	}
	wg.Wait()
}

func (s *Syncer) syncUserData(uc *UserContainer) {
	if uc.Status != StatusRunning {
		return
	}

	client := s.bbgoClient(uc.UserID)
	if err := client.Ping(); err != nil {
		log.Printf("sync user %s: bbgo api not reachable: %v", uc.UserID, err)
		return
	}

	if s.creds != nil {
		s.markCredentialsVerified(uc)
	}

	s.syncOrdersViaAPI(uc.UserID, client)
	s.syncTradesViaAPI(uc.UserID, client)
}

func (s *Syncer) getCursor(userID, table string) int64 {
	resp, err := s.supabaseRequest("GET", "sync_cursors?user_id=eq."+userID+"&table_name=eq."+table+"&select=last_gid", nil)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var rows []struct {
		LastGID int64 `json:"last_gid"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&rows); err != nil || len(rows) == 0 {
		return 0
	}
	return rows[0].LastGID
}

func (s *Syncer) updateCursor(userID, table string, lastGID int64) {
	payload, _ := json.Marshal(map[string]interface{}{
		"user_id":    userID,
		"table_name": table,
		"last_gid":   lastGID,
	})
	resp, err := s.supabaseRequest("POST", "sync_cursors?on_conflict=user_id,table_name", payload)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (s *Syncer) syncOrdersViaAPI(userID string, client *BBGoClient) {
	lastGID := s.getCursor(userID, "sync_orders")
	orders, err := client.GetAllClosedOrders("", "", lastGID)
	if err != nil {
		log.Printf("sync orders via api for user %s failed: %v", userID, err)
		return
	}

	if len(orders) == 0 {
		return
	}

	rows := make([]map[string]interface{}, len(orders))
	for i, o := range orders {
		rows[i] = map[string]interface{}{
			"user_id":  userID,
			"bot_id":   userID,
			"order_id": json.Number(formatUint(o.OrderID)),
			"symbol":   o.Symbol,
			"side":     o.Side,
			"price":    o.Price,
			"quantity": o.Quantity,
			"status":   o.Status,
			"type":     o.Type,
			"executed_quantity": o.ExecutedQuantity,
			"creation_time": o.CreationTime,
		}
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		log.Printf("marshal orders for user %s: %v", userID, err)
		return
	}

	resp, err := s.supabaseRequest("POST", "sync_orders?on_conflict=user_id,order_id", payload)
	if err != nil {
		log.Printf("sync orders for user %s failed: %v", userID, err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("sync orders for user %s rejected (status %d), not advancing cursor", userID, resp.StatusCode)
		return
	}

	maxGID := lastGID
	for _, o := range orders {
		if int64(o.GID) > maxGID {
			maxGID = int64(o.GID)
		}
	}
	s.updateCursor(userID, "sync_orders", maxGID)
	log.Printf("synced %d orders for user %s via api (cursor %d -> %d)", len(orders), userID, lastGID, maxGID)
}

func (s *Syncer) syncTradesViaAPI(userID string, client *BBGoClient) {
	lastGID := s.getCursor(userID, "sync_trades")
	trades, err := client.GetAllTradesFrom("", "", lastGID)
	if err != nil {
		log.Printf("sync trades via api for user %s failed: %v", userID, err)
		return
	}

	if len(trades) == 0 {
		return
	}

	rows := make([]map[string]interface{}, len(trades))
	for i, t := range trades {
		rows[i] = map[string]interface{}{
			"user_id":      userID,
			"bot_id":       userID,
			"trade_id":     json.Number(formatUint(t.ID)),
			"order_id":     json.Number(formatUint(t.OrderID)),
			"symbol":       t.Symbol,
			"side":         t.Side,
			"price":        t.Price,
			"quantity":     t.Quantity,
			"fee":          t.Fee,
			"fee_currency": t.FeeCurrency,
			"quote_quantity": t.QuoteQuantity,
			"traded_at":      t.TradedAt,
		}
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		log.Printf("marshal trades for user %s: %v", userID, err)
		return
	}

	resp, err := s.supabaseRequest("POST", "sync_trades?on_conflict=user_id,trade_id", payload)
	if err != nil {
		log.Printf("sync trades for user %s failed: %v", userID, err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("sync trades for user %s rejected (status %d), not advancing cursor", userID, resp.StatusCode)
		return
	}

	maxGID := lastGID
	for _, t := range trades {
		if t.GID > maxGID {
			maxGID = t.GID
		}
	}
	s.updateCursor(userID, "sync_trades", maxGID)
	log.Printf("synced %d trades for user %s via api (cursor %d -> %d)", len(trades), userID, lastGID, maxGID)
}

func (s *Syncer) SyncCredential(cred ExchangeCredential) {
	if s.creds == nil {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"user_id":              cred.UserID,
		"exchange":             cred.Exchange,
		"api_key_encrypted":    cred.APIKeyEncrypted,
		"api_secret_encrypted": cred.APISecretEncrypted,
		"passphrase_encrypted": cred.PassphraseEncrypted,
		"is_testnet":           cred.IsTestnet,
		"is_verified":          cred.IsVerified,
	})
	if err != nil {
		log.Printf("marshal credential for user %s: %v", cred.UserID, err)
		return
	}

	resp, err := s.supabaseRequest("POST", "exchange_credentials?on_conflict=user_id,exchange", payload)
	if err != nil {
		log.Printf("sync credential for user %s %s: %v", cred.UserID, cred.Exchange, err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("sync credential for user %s %s skipped (status %d)", cred.UserID, cred.Exchange, resp.StatusCode)
	} else {
		log.Printf("synced credential %s for user %s", cred.Exchange, cred.UserID)
	}
}

func (s *Syncer) markCredentialsVerified(uc *UserContainer) {
	creds, err := s.creds.List(uc.UserID)
	if err != nil {
		return
	}
	for _, c := range creds {
		if c.IsVerified {
			continue
		}
		exchanges := []string{}
		for _, strat := range uc.Strategies {
			if strat.CrossExchange {
				for _, sr := range strat.Sessions {
					exchanges = append(exchanges, sr.Exchange)
				}
			} else if strat.Exchange != "" {
				exchanges = append(exchanges, strat.Exchange)
			}
		}
		for _, ex := range exchanges {
			if ex == c.Exchange {
				c.IsVerified = true
				s.creds.Update(uc.UserID, c)
				log.Printf("credential %s for user %s marked as verified", c.Exchange, uc.UserID)
				break
			}
		}
	}
}

func (s *Syncer) DeleteCredential(userID, exchange string) {
	if s.creds == nil {
		return
	}
	path := "exchange_credentials?user_id=eq." + userID + "&exchange=eq." + exchange
	resp, err := s.supabaseRequest("DELETE", path, nil)
	if err != nil {
		log.Printf("delete credential for user %s %s: %v", userID, exchange, err)
		return
	}
	resp.Body.Close()
	log.Printf("deleted credential %s for user %s from supabase", exchange, userID)
}
