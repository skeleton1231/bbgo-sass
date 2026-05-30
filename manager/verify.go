package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type VerifyResult struct {
	Verified bool   `json:"verified"`
	Error    string `json:"error,omitempty"`
}

var exchangeBaseURLs = map[string]struct{ live, testnet string }{
	"binance":  {"https://api.binance.com", "https://testnet.binance.vision"},
	"bybit":    {"https://api.bybit.com", "https://api-testnet.bybit.com"},
	"bitget":   {"https://api.bitget.com", "https://api.bitget.com"},
	"okex":     {"https://www.okx.com", "https://www.okx.com"},
	"kucoin":   {"https://api.kucoin.com", "https://openapi-sandbox.kucoin.com"},
	"max":      {"https://max-api.maicoin.com", "https://max-api.maicoin.com"},
	"coinbase": {"https://api.exchange.coinbase.com", "https://api-public.sandbox.exchange.coinbase.com"},
	"bitfinex": {"https://api.bitfinex.com", "https://api.bitfinex.com"},
}

func exchangeHasVerifier(exchange string) bool {
	switch exchange {
	case "binance", "bybit", "bitget":
		return true
	default:
		return false
	}
}

func verifyCredential(exchange, apiKey, apiSecret, passphrase string, isTestnet bool) VerifyResult {
	urls, ok := exchangeBaseURLs[exchange]
	if !ok {
		return VerifyResult{Error: fmt.Sprintf("unsupported exchange: %s", exchange)}
	}

	baseURL := urls.live
	if isTestnet {
		baseURL = urls.testnet
	}

	switch exchange {
	case "binance":
		return verifyBinance(baseURL, apiKey, apiSecret)
	case "bybit":
		return verifyBybit(baseURL, apiKey, apiSecret)
	case "bitget":
		return verifyBitget(baseURL, apiKey, apiSecret, passphrase)
	default:
		return VerifyResult{Error: fmt.Sprintf("verification not implemented for %s", exchange)}
	}
}

func verifyBinance(baseURL, apiKey, apiSecret string) VerifyResult {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	query := "timestamp=" + ts
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(query))
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("GET", baseURL+"/api/v3/account?"+query+"&signature="+sig, nil)
	if err != nil {
		return VerifyResult{Error: err.Error()}
	}
	req.Header.Set("X-MBX-APIKEY", apiKey)

	return doVerifyRequest(req)
}

func verifyBybit(baseURL, apiKey, apiSecret string) VerifyResult {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recvWindow := "5000"
	paramStr := ts + apiKey + recvWindow + "coin=USDT"
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(paramStr))
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("GET", baseURL+"/v5/account/wallet-balance?coin=USDT", nil)
	if err != nil {
		return VerifyResult{Error: err.Error()}
	}
	req.Header.Set("X-BAPI-API-KEY", apiKey)
	req.Header.Set("X-BAPI-SIGN", sig)
	req.Header.Set("X-BAPI-TIMESTAMP", ts)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)

	return doVerifyRequest(req)
}

func verifyBitget(baseURL, apiKey, apiSecret, passphrase string) VerifyResult {
	if passphrase == "" {
		return VerifyResult{Error: "Bitget requires a passphrase"}
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := "/api/v2/spot/account/assets?coin=USDT"
	preHash := ts + "GET" + path
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(preHash))
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return VerifyResult{Error: err.Error()}
	}
	req.Header.Set("ACCESS-KEY", apiKey)
	req.Header.Set("ACCESS-SIGN", sig)
	req.Header.Set("ACCESS-TIMESTAMP", ts)
	req.Header.Set("ACCESS-PASSPHRASE", passphrase)
	req.Header.Set("locale", "en-US")

	return doVerifyRequest(req)
}

var verifyHTTPClient = &http.Client{Timeout: 10 * time.Second}

func doVerifyRequest(req *http.Request) VerifyResult {
	resp, err := verifyHTTPClient.Do(req)
	if err != nil {
		return VerifyResult{Error: fmt.Sprintf("connection failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return VerifyResult{Verified: true}
	}
	body, _ := io.ReadAll(resp.Body)
	return VerifyResult{Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 100))}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
