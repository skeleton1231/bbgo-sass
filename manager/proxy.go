package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type BotProxy struct {
	cm *ContainerManager
}

func NewBotProxy(cm *ContainerManager) *BotProxy {
	return &BotProxy{cm: cm}
}

func (bp *BotProxy) ProxyToBot(w http.ResponseWriter, r *http.Request, userID string) {
	name := bp.cm.containerName(userID)
	targetPath := strings.TrimPrefix(r.URL.Path, "/api/bbgo/"+userID)
	if targetPath == "" || targetPath == "/" {
		targetPath = "/"
	}

	targetURL := fmt.Sprintf("http://%s:%d/api%s", name, bp.cm.cfg.BBGOPort, targetPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header.Clone()

	resp, err := http.DefaultClient.Do(proxyReq)
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
