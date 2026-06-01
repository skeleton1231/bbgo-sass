package main

import "encoding/json"

// --- API response envelopes ---

type healthResponse struct {
	Status  string `json:"status"`
	Users   int    `json:"users"`
	Running int    `json:"running"`
}

type containerStatusResponse struct {
	UserID     string                    `json:"user_id"`
	Containers map[string]*containerInfo `json:"containers"`
}

type strategyCreatedResponse struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
	Mode   string `json:"mode"`
}

type sessionsResponse struct {
	Sessions []BBGoSession `json:"sessions"`
}

type sessionDetailResponse struct {
	Session BBGoSession `json:"session"`
}

type tradesResponse struct {
	Trades []BBGoTrade `json:"trades"`
}

type ordersResponse struct {
	Orders []BBGoOrder `json:"orders"`
}

type balancesResponse struct {
	Balances map[string]BBGoBalance `json:"balances"`
}

type symbolsResponse struct {
	Symbols []string `json:"symbols"`
}

type assetsResponse struct {
	Assets map[string]BBGoAsset `json:"assets"`
}

type bbgoStrategiesResponse struct {
	Strategies []BBGoStrategyState `json:"strategies"`
}

type logsResponse struct {
	Logs string `json:"logs"`
}

type backtestResultResponse struct {
	Output string `json:"output"`
}

type backtestSyncResponse struct {
	Exchange string       `json:"exchange"`
	Synced   []syncResult `json:"synced"`
}

type syncResult struct {
	Symbol string `json:"symbol"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type backtestSyncStatusResponse struct {
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Modified  string `json:"modified,omitempty"`
}

type backtestJobsResponse struct {
	Jobs []*BacktestJob `json:"jobs"`
}

type backtestSubmitResponse struct {
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	NeedSync bool   `json:"need_sync"`
}

// --- Ticker & Klines ---

type tickerData struct {
	Symbol string  `json:"symbol"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type tickerResponse struct {
	Ticker tickerData `json:"ticker"`
}

type klineEntry struct {
	Time        int64  `json:"time"`
	Open        string `json:"open"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Close       string `json:"close"`
	Volume      string `json:"volume"`
	QuoteVolume string `json:"quoteVolume"`
	Closed      bool   `json:"closed"`
}

type klinesResponse struct {
	Klines []klineEntry `json:"klines"`
}

// --- Credentials ---

type credentialResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Exchange    string `json:"exchange"`
	IsTestnet   bool   `json:"is_testnet"`
	IsVerified  bool   `json:"is_verified"`
	VerifyError string `json:"verify_error,omitempty"`
}

// --- Notifications ---

type notifConfigResponse struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Enabled bool             `json:"enabled"`
	Rules   NotificationRule `json:"rules"`
}

// --- Container status ---

type containerInfo struct {
	UserID     string               `json:"user_id"`
	Mode       string               `json:"mode"`
	Status     string               `json:"status"`
	Strategies []BBGoStrategyState  `json:"strategies,omitempty"`
}

// --- Trading volume passthrough ---

type tradingVolumeResponse struct {
	TradingVolumes json.RawMessage `json:"tradingVolumes"`
}

// --- Account passthrough ---

type accountResponse struct {
	Account json.RawMessage `json:"account"`
}

// --- Bots ---

type botsResponse struct {
	Bots []Bot `json:"bots"`
}

// --- Status messages ---

type statusMessage struct {
	Status string `json:"status"`
}

type statusStopped struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type statusRestarting struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
	Mode   string `json:"mode"`
}

type statusStoppedUser struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
	Mode   string `json:"mode"`
}
