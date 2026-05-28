package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
)

type Syncer struct {
	users           *UserContainerManager
	cfg             *Config
	container       *ContainerManager
	creds           *CredentialStore
	client          *http.Client
	pool            *pool.Pool
	newBBGoClientFn func(baseURL string) *BBGoClient
}

func NewSyncer(users *UserContainerManager, cfg *Config, cm *ContainerManager, p *pool.Pool) *Syncer {
	return &Syncer{
		users:     users,
		cfg:       cfg,
		container: cm,
		client:    &http.Client{Timeout: 10 * time.Second},
		pool:      p,
	}
}

func NewSyncerWithCreds(users *UserContainerManager, cfg *Config, cm *ContainerManager, creds *CredentialStore, p *pool.Pool) *Syncer {
	s := NewSyncer(users, cfg, cm, p)
	s.creds = creds
	return s
}

func (s *Syncer) bbgoClient(userID, mode string) *BBGoClient {
	if s.newBBGoClientFn != nil {
		return s.newBBGoClientFn(s.container.APIURL(userID, mode))
	}
	return NewBBGoClient(s.container.APIURL(userID, mode))
}

func readBodyHint(resp *http.Response) string {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return string(b)
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
		return nil, fmt.Errorf("load users from supabase: unexpected status %d", resp.StatusCode)
	}

	var raw []struct {
		UserID     string          `json:"user_id"`
		Mode       string          `json:"mode"`
		Status     string          `json:"status"`
		Strategies json.RawMessage `json:"strategies"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPnLResponseBody)).Decode(&raw); err != nil {
		return nil, err
	}

	users := make([]*UserContainer, len(raw))
	for i, r := range raw {
		uc := &UserContainer{
			UserID: r.UserID,
			Mode:   r.Mode,
			Status: r.Status,
		}
		if uc.Mode == "" {
			uc.Mode = ModeLive
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
		"mode":       uc.Mode,
		"status":     uc.Status,
		"strategies": uc.Strategies,
	})
	if err != nil {
		log.Printf("marshal user %s: %v", uc.UserID, err)
		return
	}

	resp, err := s.supabaseRequest("POST", "user_containers?on_conflict=user_id,mode", payload)
	if err != nil {
		log.Printf("upsert user %s failed: %v", uc.UserID, err)
		return
	}
	if resp.StatusCode >= 400 {
		log.Printf("upsert user %s rejected (status %d): %s", uc.UserID, resp.StatusCode, readBodyHint(resp))
	}
	resp.Body.Close()
}

func (s *Syncer) SyncUser(userID, mode string) {
	uc, ok := s.users.Get(userID, mode)
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
		if uc.Status != StatusRunning {
			continue
		}
		uc := uc
		if err := s.pool.Submit(func() { s.syncUserData(uc) }); err != nil {
			log.Printf("sync pool submit for user %s: %v", uc.UserID, err)
		}
	}
	s.pool.Wait()
}

func (s *Syncer) syncUserData(uc *UserContainer) {
	if uc.Status != StatusRunning {
		return
	}

	client := s.bbgoClient(uc.UserID, uc.Mode)
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPnLResponseBody)).Decode(&rows); err != nil || len(rows) == 0 {
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
	if resp.StatusCode >= 400 {
		log.Printf("update cursor %s/%s failed (status %d): %s", userID, table, resp.StatusCode, readBodyHint(resp))
	}
	resp.Body.Close()
}

func (s *Syncer) syncOrdersViaAPI(userID string, client *BBGoClient) {
	cursor := s.getCursor(userID, "sync_orders")

	if cursor == 0 {
		s.fullSyncOrders(userID, client)
		return
	}

	// Incremental sync: bbgo's /api/orders/closed uses DESC ordering (gid < :gid),
	// which paginates backward. To get NEW orders, fetch without cursor (gid=0)
	// which returns the latest 500, then filter to only orders with GID > saved cursor.
	orders, err := client.GetClosedOrders("", "", 0)
	if err != nil {
		log.Printf("sync orders via api for user %s failed: %v", userID, err)
		return
	}
	if len(orders) == 0 {
		return
	}

	var newOrders []BBGoOrder
	for _, o := range orders {
		if int64(o.GID) > cursor {
			newOrders = append(newOrders, o)
		}
	}

	if len(newOrders) == 0 {
		return
	}

	// If the raw batch is full (500), there may be more orders beyond this
	// page. Fall back to full sync to ensure nothing is missed.
	if len(orders) == syncPageSize {
		s.fullSyncOrders(userID, client)
		return
	}

	maxGID := cursor
	for _, o := range newOrders {
		if int64(o.GID) > maxGID {
			maxGID = int64(o.GID)
		}
	}

	if err := s.upsertOrders(userID, newOrders); err != nil {
		return
	}
	s.updateCursor(userID, "sync_orders", maxGID)
	log.Printf("synced %d orders for user %s via api (cursor %d -> %d)", len(newOrders), userID, cursor, maxGID)
}

// fullSyncOrders paginates backward through all orders for initial sync (cursor=0).
// The bbgo API returns orders in GID DESC order with LIMIT 500, so we paginate
// backward and save the global max GID as the cursor.
func (s *Syncer) fullSyncOrders(userID string, client *BBGoClient) {
	var globalMaxGID int64
	totalSynced := 0
	cursor := int64(0)

	for {
		orders, err := client.GetClosedOrders("", "", cursor)
		if err != nil {
			log.Printf("sync orders via api for user %s failed: %v", userID, err)
			return
		}
		if len(orders) == 0 {
			break
		}

		if err := s.upsertOrders(userID, orders); err != nil {
			log.Printf("full sync orders for user %s aborted: %v", userID, err)
			return
		}
		totalSynced += len(orders)

		for _, o := range orders {
			if int64(o.GID) > globalMaxGID {
				globalMaxGID = int64(o.GID)
			}
		}

		// Advance cursor to smallest GID in batch (last element since DESC)
		cursor = int64(orders[len(orders)-1].GID)
		if cursor <= 0 {
			break
		}

		if len(orders) < syncPageSize {
			break
		}
	}

	if totalSynced > 0 {
		s.updateCursor(userID, "sync_orders", globalMaxGID)
		log.Printf("full synced %d orders for user %s via api (cursor 0 -> %d)", totalSynced, userID, globalMaxGID)
	}
}

func (s *Syncer) upsertOrders(userID string, orders []BBGoOrder) error {
	rows := make([]map[string]interface{}, len(orders))
	for i, o := range orders {
		rows[i] = map[string]interface{}{
			"user_id":           userID,
			"order_id":          json.Number(formatUint(o.OrderID)),
			"symbol":            o.Symbol,
			"side":              o.Side,
			"price":             o.Price,
			"quantity":          o.Quantity,
			"status":            o.Status,
			"type":              o.Type,
			"executed_quantity": o.ExecutedQuantity,
			"creation_time":     o.CreationTime,
		}
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("marshal orders for user %s: %w", userID, err)
	}

	resp, err := s.supabaseRequest("POST", "sync_orders?on_conflict=user_id,order_id", payload)
	if err != nil {
		return fmt.Errorf("sync orders for user %s: %w", userID, err)
	}
	if resp.StatusCode >= 400 {
		hint := readBodyHint(resp)
		resp.Body.Close()
		return fmt.Errorf("sync orders for user %s rejected (status %d): %s", userID, resp.StatusCode, hint)
	}
	resp.Body.Close()
	return nil
}

func (s *Syncer) syncTradesViaAPI(userID string, client *BBGoClient) {
	cursor := s.getCursor(userID, "sync_trades")
	totalSynced := 0
	startCursor := cursor

	for {
		trades, err := client.GetTrades("", "", cursor)
		if err != nil {
			log.Printf("sync trades via api for user %s failed: %v", userID, err)
			return
		}
		if len(trades) == 0 {
			break
		}

		rows := make([]map[string]interface{}, len(trades))
		for i, t := range trades {
			rows[i] = map[string]interface{}{
				"user_id":        userID,
				"trade_id":       json.Number(formatUint(t.ID)),
				"order_id":       json.Number(formatUint(t.OrderID)),
				"symbol":         t.Symbol,
				"side":           t.Side,
				"price":          t.Price,
				"quantity":       t.Quantity,
				"fee":            t.Fee,
				"fee_currency":   t.FeeCurrency,
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
		if resp.StatusCode >= 400 {
			log.Printf("sync trades for user %s rejected (status %d): %s", userID, resp.StatusCode, readBodyHint(resp))
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		maxGID := cursor
		for _, t := range trades {
			if t.GID > maxGID {
				maxGID = t.GID
			}
		}
		cursor = maxGID
		s.updateCursor(userID, "sync_trades", cursor)
		totalSynced += len(trades)

		if len(trades) < syncPageSize {
			break
		}
	}

	if totalSynced > 0 {
		log.Printf("synced %d trades for user %s via api (cursor %d -> %d)", totalSynced, userID, startCursor, cursor)
	}
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

	resp, err := s.supabaseRequest("POST", "exchange_credentials?on_conflict=user_id,exchange,is_testnet", payload)
	if err != nil {
		log.Printf("sync credential for user %s %s: %v", cred.UserID, cred.Exchange, err)
		return
	}
	if resp.StatusCode >= 400 {
		log.Printf("sync credential for user %s %s skipped (status %d): %s", cred.UserID, cred.Exchange, resp.StatusCode, readBodyHint(resp))
	} else {
		log.Printf("synced credential %s for user %s", cred.Exchange, cred.UserID)
	}
	resp.Body.Close()
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

const pnlPageSize = 1000

const maxPnLResponseBody = 2 << 20 // 2 MiB

type pnlTradeRow struct {
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Fee      string `json:"fee"`
	TradedAt string `json:"traded_at"`
}

func (s *Syncer) GetTradesForPnL(userID string) ([]BBGoTrade, error) {
	var allTrades []BBGoTrade
	offset := 0

	for {
		params := url.Values{}
		params.Set("user_id", "eq."+userID)
		params.Set("order", "traded_at.asc")
		params.Set("select", "symbol,side,price,quantity,fee,traded_at")
		params.Set("offset", fmt.Sprintf("%d", offset))
		params.Set("limit", fmt.Sprintf("%d", pnlPageSize))
		path := "sync_trades?" + params.Encode()
		resp, err := s.supabaseRequest("GET", path, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch trades for pnl: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetch trades for pnl: status %d", resp.StatusCode)
		}

		var rows []pnlTradeRow
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxPnLResponseBody)).Decode(&rows); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode trades for pnl: %w", err)
		}
		resp.Body.Close()

		for _, r := range rows {
			allTrades = append(allTrades, BBGoTrade{
				Symbol: r.Symbol, Side: r.Side, Price: flexString(r.Price),
				Quantity: flexString(r.Quantity), Fee: flexString(r.Fee), TradedAt: r.TradedAt,
			})
		}

		if len(rows) < pnlPageSize {
			break
		}
		offset += len(rows)
	}
	return allTrades, nil
}
