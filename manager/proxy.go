package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type BotProxy struct {
	cm     *ContainerManager
	client *http.Client
}

func NewBotProxy(cm *ContainerManager) *BotProxy {
	transport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		Proxy:      func(_ *http.Request) (*url.URL, error) { return nil, nil },
	}
	return &BotProxy{
		cm: cm,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

func (bp *BotProxy) ProxyToBot(w http.ResponseWriter, r *http.Request, userID string) {
	targetPath := strings.TrimPrefix(r.URL.Path, "/api/bbgo/"+userID)
	if targetPath == "" || targetPath == "/" {
		targetPath = "/"
	}

	baseURL := bp.cm.APIURL(userID)
	targetURL := fmt.Sprintf("%s/api%s", baseURL, targetPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header.Clone()

	resp, err := bp.client.Do(proxyReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"error":   "bot api unavailable",
			"user_id": userID,
			"details": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, io.LimitReader(resp.Body, 10<<20))
}
