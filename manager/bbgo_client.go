package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type BBGoClient struct {
	baseURL string
	client  *http.Client
	ctx     context.Context
}

func NewBBGoClient(baseURL string) *BBGoClient {
	return &BBGoClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
		ctx:     context.Background(),
	}
}

func (c *BBGoClient) WithContext(ctx context.Context) *BBGoClient {
	return &BBGoClient{baseURL: c.baseURL, client: c.client, ctx: ctx}
}

func (c *BBGoClient) get(path string, result interface{}) error {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("bbgo api %s: %w", path, err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("bbgo api %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("bbgo api %s: status %d: %s", path, resp.StatusCode, string(body))
	}

	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(result)
}

// Ping checks if the bbgo container API is responding.
func (c *BBGoClient) Ping() error {
	var resp struct {
		Message string `json:"message"`
	}
	return c.get("/api/ping", &resp)
}

type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = flexString(n.String())
		return nil
	}
	return fmt.Errorf("flexString: cannot unmarshal %s", string(data))
}

type BBGoTrade struct {
	GID           int64      `json:"gid"`
	ID            uint64     `json:"id"`
	OrderID       uint64     `json:"orderID"`
	OrderUUID     string     `json:"orderUUID,omitempty"`
	Exchange      string     `json:"exchange"`
	Symbol        string     `json:"symbol"`
	Side          string     `json:"side"`
	Price         flexString `json:"price"`
	Quantity      flexString `json:"quantity"`
	QuoteQuantity flexString `json:"quoteQuantity"`
	IsBuyer       bool       `json:"isBuyer"`
	IsMaker       bool       `json:"isMaker"`
	TradedAt      string     `json:"tradedAt"`
	Fee           flexString `json:"fee"`
	FeeCurrency   string     `json:"feeCurrency"`
}

type BBGoTradesResponse struct {
	Trades []BBGoTrade `json:"trades"`
}

func (c *BBGoClient) GetTrades(exchange, symbol string, lastGID int64) ([]BBGoTrade, error) {
	q := url.Values{"gid": {strconv.FormatInt(lastGID, 10)}}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if symbol != "" {
		q.Set("symbol", symbol)
	}

	var resp BBGoTradesResponse
	if err := c.get("/api/trades?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Trades, nil
}

const syncPageSize = 500

// GetAllTrades paginates through trades using the GID cursor.
func (c *BBGoClient) GetAllTrades(exchange, symbol string) ([]BBGoTrade, error) {
	return c.GetAllTradesFrom(exchange, symbol, 0)
}

func (c *BBGoClient) GetAllTradesFrom(exchange, symbol string, lastGID int64) ([]BBGoTrade, error) {
	var all []BBGoTrade
	cursor := lastGID
	for {
		trades, err := c.GetTrades(exchange, symbol, cursor)
		if err != nil {
			return nil, err
		}
		if len(trades) == 0 {
			break
		}
		all = append(all, trades...)
		maxGID := cursor
		for _, t := range trades {
			if t.GID > maxGID {
				maxGID = t.GID
			}
		}
		cursor = maxGID
		if len(trades) < syncPageSize {
			break
		}
	}
	return all, nil
}

type BBGoOrder struct {
	GID              uint64     `json:"gid"`
	OrderID          uint64     `json:"orderID"`
	UUID             string     `json:"uuid,omitempty"`
	ClientOrderID    string     `json:"clientOrderID,omitempty"`
	Exchange         string     `json:"exchange"`
	Symbol           string     `json:"symbol"`
	Side             string     `json:"side"`
	Type             string     `json:"orderType"`
	Price            flexString `json:"price"`
	Quantity         flexString `json:"quantity"`
	ExecutedQuantity flexString `json:"executedQuantity"`
	Status           string     `json:"status"`
	StopPrice        flexString `json:"stopPrice,omitempty"`
	CreationTime     string     `json:"creationTime"`
	IsWorking        bool       `json:"isWorking"`
}

type BBGoOrdersResponse struct {
	Orders []BBGoOrder `json:"orders"`
}

func (c *BBGoClient) GetClosedOrders(exchange, symbol string, lastGID int64) ([]BBGoOrder, error) {
	q := url.Values{"gid": {strconv.FormatInt(lastGID, 10)}}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if symbol != "" {
		q.Set("symbol", symbol)
	}

	var resp BBGoOrdersResponse
	if err := c.get("/api/orders/closed?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

type BBGoSession struct {
	Name         string `json:"name"`
	ExchangeName string `json:"exchange"`
}

type BBGoSessionsResponse struct {
	Sessions []BBGoSession `json:"sessions"`
}

func (c *BBGoClient) GetSessions() ([]BBGoSession, error) {
	var resp BBGoSessionsResponse
	if err := c.get("/api/sessions", &resp); err != nil {
		return nil, err
	}
	return resp.Sessions, nil
}

type BBGoSessionDetail struct {
	Session BBGoSession `json:"session"`
}

func (c *BBGoClient) GetSession(name string) (*BBGoSession, error) {
	var resp BBGoSessionDetail
	if err := c.get("/api/sessions/"+url.PathEscape(name), &resp); err != nil {
		return nil, err
	}
	return &resp.Session, nil
}

type BBGoSessionTradesResponse struct {
	Trades []BBGoTrade `json:"trades"`
}

func (c *BBGoClient) GetSessionTrades(session string) ([]BBGoTrade, error) {
	var resp BBGoSessionTradesResponse
	if err := c.get("/api/sessions/"+url.PathEscape(session)+"/trades", &resp); err != nil {
		return nil, err
	}
	return resp.Trades, nil
}

func (c *BBGoClient) GetSessionOpenOrders(session string) ([]BBGoOrder, error) {
	var resp BBGoOrdersResponse
	if err := c.get("/api/sessions/"+url.PathEscape(session)+"/open-orders", &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

type BBGoBalance struct {
	Currency  string      `json:"currency"`
	Available json.Number `json:"available"`
	Locked    json.Number `json:"locked"`
}

type BBGoAccountResponse struct {
	Account interface{} `json:"account"`
}

func (c *BBGoClient) GetSessionAccount(session string) (interface{}, error) {
	var resp BBGoAccountResponse
	if err := c.get("/api/sessions/"+url.PathEscape(session)+"/account", &resp); err != nil {
		return nil, err
	}
	return resp.Account, nil
}

type BBGoBalancesResponse struct {
	Balances map[string]BBGoBalance `json:"balances"`
}

func (c *BBGoClient) GetSessionBalances(session string) (map[string]BBGoBalance, error) {
	var resp BBGoBalancesResponse
	if err := c.get("/api/sessions/"+url.PathEscape(session)+"/account/balances", &resp); err != nil {
		return nil, err
	}
	return resp.Balances, nil
}

type BBGoSymbolsResponse struct {
	Symbols []string `json:"symbols"`
}

func (c *BBGoClient) GetSessionSymbols(session string) ([]string, error) {
	var resp BBGoSymbolsResponse
	if err := c.get("/api/sessions/"+url.PathEscape(session)+"/symbols", &resp); err != nil {
		return nil, err
	}
	return resp.Symbols, nil
}

type BBGoAsset struct {
	Currency      string      `json:"currency"`
	Total         json.Number `json:"total"`
	Available     json.Number `json:"available"`
	Locked        json.Number `json:"lock"`
	Borrowed      json.Number `json:"borrowed"`
	NetAsset      json.Number `json:"netAsset"`
	NetAssetInUSD json.Number `json:"netAssetInUSD"`
	NetAssetInBTC json.Number `json:"netAssetInBTC"`
	PriceInUSD    json.Number `json:"priceInUSD"`
}

type BBGoAssetsResponse struct {
	Assets map[string]BBGoAsset `json:"assets"`
}

func (c *BBGoClient) GetAssets() (map[string]BBGoAsset, error) {
	var resp BBGoAssetsResponse
	if err := c.get("/api/assets", &resp); err != nil {
		return nil, err
	}
	return resp.Assets, nil
}

type BBGoStrategyState map[string]interface{}

type BBGoStrategiesResponse struct {
	Strategies []BBGoStrategyState `json:"strategies"`
}

func (c *BBGoClient) GetStrategies() ([]BBGoStrategyState, error) {
	var resp BBGoStrategiesResponse
	if err := c.get("/api/strategies/single", &resp); err != nil {
		return nil, err
	}
	return resp.Strategies, nil
}

type BBGoTradingVolumeResponse struct {
	TradingVolumes interface{} `json:"tradingVolumes"`
}

func (c *BBGoClient) GetTradingVolume(period, segment string) (interface{}, error) {
	path := "/api/trading-volume"
	q := url.Values{}
	if period != "" {
		q.Set("period", period)
	}
	if segment != "" {
		q.Set("segment", segment)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var resp BBGoTradingVolumeResponse
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return resp.TradingVolumes, nil
}

func formatUint(v uint64) string {
	return strconv.FormatUint(v, 10)
}
