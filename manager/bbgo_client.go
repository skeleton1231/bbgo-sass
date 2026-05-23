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
}

func NewBBGoClient(baseURL string) *BBGoClient {
	return &BBGoClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *BBGoClient) get(path string, result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
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

	return json.NewDecoder(resp.Body).Decode(result)
}

// Ping checks if the bbgo container API is responding.
func (c *BBGoClient) Ping() error {
	var resp struct {
		Message string `json:"message"`
	}
	return c.get("/api/ping", &resp)
}

type BBGoTrade struct {
	GID            int64  `json:"gid"`
	ID             uint64 `json:"id"`
	OrderID        uint64 `json:"orderID"`
	OrderUUID      string `json:"orderUUID,omitempty"`
	Exchange       string `json:"exchange"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	Price          string `json:"price"`
	Quantity       string `json:"quantity"`
	QuoteQuantity  string `json:"quoteQuantity"`
	IsBuyer        bool   `json:"isBuyer"`
	IsMaker        bool   `json:"isMaker"`
	TradedAt       string `json:"tradedAt"`
	Fee            string `json:"fee"`
	FeeCurrency    string `json:"feeCurrency"`
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
	if err := c.get("/api/trades?" + q.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Trades, nil
}

type BBGoOrder struct {
	GID              uint64 `json:"gid"`
	OrderID          uint64 `json:"orderID"`
	UUID             string `json:"uuid,omitempty"`
	ClientOrderID    string `json:"clientOrderID,omitempty"`
	Exchange         string `json:"exchange"`
	Symbol           string `json:"symbol"`
	Side             string `json:"side"`
	Type             string `json:"orderType"`
	Price            string `json:"price"`
	Quantity         string `json:"quantity"`
	ExecutedQuantity string `json:"executedQuantity"`
	Status           string `json:"status"`
	StopPrice        string `json:"stopPrice,omitempty"`
	CreationTime     string `json:"creationTime"`
	IsWorking        bool   `json:"isWorking"`
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
	if err := c.get("/api/orders/closed?" + q.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

type BBGoSession struct {
	Name         string `json:"name"`
	ExchangeName string `json:"exchangeName"`
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
	Currency  string `json:"currency"`
	Available string `json:"available"`
	Locked    string `json:"locked"`
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
	Currency      string `json:"currency"`
	Total         string `json:"total"`
	Available     string `json:"available"`
	Locked        string `json:"lock"`
	Borrowed      string `json:"borrowed"`
	NetAsset      string `json:"netAsset"`
	NetAssetInUSD string `json:"netAssetInUSD"`
	NetAssetInBTC string `json:"netAssetInBTC"`
	PriceInUSD    string `json:"priceInUSD"`
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

type BBGoStrategyState struct {
	Strategy string `json:"strategy"`
}

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
