package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (c *BBGoClient) get(path string, result any) error {
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
	GID             int64      `json:"gid"`
	ID              uint64     `json:"id"`
	OrderID         uint64     `json:"orderID"`
	OrderUUID       string     `json:"orderUUID,omitempty"`
	Exchange        string     `json:"exchange"`
	Symbol          string     `json:"symbol"`
	Side            string     `json:"side"`
	Price           flexString `json:"price"`
	Quantity        flexString `json:"quantity"`
	QuoteQuantity   flexString `json:"quoteQuantity"`
	IsBuyer         bool       `json:"isBuyer"`
	IsMaker         bool       `json:"isMaker"`
	TradedAt        string     `json:"tradedAt"`
	Fee             flexString `json:"fee"`
	FeeCurrency     string     `json:"feeCurrency"`
	StrategyID      string     `json:"strategyID,omitempty"`
	NetPosition     float64    `json:"netPosition,omitempty"`
}

type BBGoTradesResponse struct {
	Trades []BBGoTrade `json:"trades"`
}

type BBGoSession struct {
	Name         string `json:"name"`
	ExchangeName string `json:"exchangeName"`
}

type BBGoSessionsResponse struct {
	Sessions []BBGoSession `json:"sessions"`
}

type BBGoSymbolsResponse struct {
	Symbols []string `json:"symbols"`
}

type BBGoStrategyState map[string]any

type BBGoStrategiesResponse struct {
	Strategies []BBGoStrategyState `json:"strategies"`
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

func (c *BBGoClient) GetSessionSymbols(session string) ([]string, error) {
	var resp BBGoSymbolsResponse
	if err := c.get("/api/sessions/"+url.PathEscape(session)+"/symbols", &resp); err != nil {
		return nil, err
	}
	return resp.Symbols, nil
}

func (c *BBGoClient) GetStrategies() ([]BBGoStrategyState, error) {
	var resp BBGoStrategiesResponse
	if err := c.get("/api/strategies/single", &resp); err != nil {
		return nil, err
	}
	return resp.Strategies, nil
}

