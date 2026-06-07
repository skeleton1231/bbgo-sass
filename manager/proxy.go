package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type BotProxy struct {
	cm          *ContainerManager
	client      *http.Client
	resolveAddr func(userID, mode, instanceID string) string
}

func NewBotProxy(cm *ContainerManager) *BotProxy {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		Proxy:                 func(_ *http.Request) (*url.URL, error) { return nil, nil },
		ResponseHeaderTimeout: 15 * time.Second,
	}
	return &BotProxy{
		cm: cm,
		resolveAddr: func(userID, mode, instanceID string) string {
			return cm.InstanceAPIURL(userID, mode, instanceID)
		},
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

func (bp *BotProxy) ProxyToInstance(w http.ResponseWriter, r *http.Request, inst *StrategyInstance) {
	targetPath := strings.TrimPrefix(r.URL.Path, "/api/bbgo/"+inst.UserID)
	if targetPath == "" || targetPath == "/" {
		targetPath = "/"
	}

	baseURL := bp.resolveAddr(inst.UserID, inst.Mode, inst.InstanceID)
	targetURL := fmt.Sprintf("%s/api%s", baseURL, targetPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header.Clone()
	proxyReq.Header.Del("X-Manager-Token")
	proxyReq.Header.Del("X-User-Id")

	resp, err := bp.client.Do(proxyReq)
	if err != nil {
		code := http.StatusBadGateway
		if r.Context().Err() != nil {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, map[string]any{
			"error":       "bot api unavailable",
			"user_id":     inst.UserID,
			"instance_id": inst.InstanceID,
			"details":     err.Error(),
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

	_, copyErr := io.Copy(w, io.LimitReader(resp.Body, 10<<20))
	if copyErr != nil {
		log.Printf("proxy copy to client for instance %s: %v", inst.InstanceID, copyErr)
	}
}
