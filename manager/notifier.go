package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type NotificationChannel struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Type       string `json:"type"` // "telegram" | "slack"
	TokenEnc   string `json:"token_enc,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
	WebhookURL string `json:"webhook_url,omitempty"`
	Enabled    bool   `json:"enabled"`
}

type NotificationRule struct {
	TradeEvents     bool `json:"trade_events"`
	OrderEvents     bool `json:"order_events"`
	ContainerHealth bool `json:"container_health"`
}

type NotificationConfig struct {
	Channel NotificationChannel `json:"channel"`
	Rules   NotificationRule    `json:"rules"`
}

type NotificationEvent struct {
	Type    string `json:"type"` // "trade", "order", "container"
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Notifier struct {
	mu        sync.RWMutex
	dir       string
	crypto    *Encryptor
	client    *http.Client
	configs   map[string][]NotificationConfig // userID -> configs
	lastSent  map[string]map[string]time.Time // userID -> eventType -> lastSent
	rateLimit time.Duration
}

func NewNotifier(dataDir string, enc *Encryptor) *Notifier {
	return &Notifier{
		dir:       dataDir,
		crypto:    enc,
		client:    &http.Client{Timeout: 10 * time.Second},
		configs:   make(map[string][]NotificationConfig),
		lastSent:  make(map[string]map[string]time.Time),
		rateLimit: 1 * time.Minute,
	}
}

func (n *Notifier) filePath(userID string) string {
	return filepath.Join(n.dir, userID, "notifications.json")
}

func (n *Notifier) loadAll(userID string) ([]NotificationConfig, error) {
	data, err := os.ReadFile(n.filePath(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var configs []NotificationConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

func (n *Notifier) saveAll(userID string, configs []NotificationConfig) error {
	dir := filepath.Dir(n.filePath(userID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(n.filePath(userID), data, 0o600)
}

func (n *Notifier) LoadUser(userID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	configs, err := n.loadAll(userID)
	if err != nil {
		return
	}
	n.configs[userID] = configs
}

func (n *Notifier) Create(userID string, cfg NotificationConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	configs := n.configs[userID]
	configs = append(configs, cfg)
	n.configs[userID] = configs
	return n.saveAll(userID, configs)
}

func (n *Notifier) List(userID string) []NotificationConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()
	configs := n.configs[userID]
	out := make([]NotificationConfig, len(configs))
	copy(out, configs)
	return out
}

func (n *Notifier) Delete(userID, id string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	configs := n.configs[userID]
	filtered := make([]NotificationConfig, 0, len(configs))
	for _, c := range configs {
		if c.Channel.ID != id {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == len(configs) {
		return fmt.Errorf("notification config %s not found", id)
	}
	n.configs[userID] = filtered
	return n.saveAll(userID, filtered)
}

func (n *Notifier) Dispatch(userID string, event NotificationEvent) bool {
	n.mu.Lock()
	configs := n.configs[userID]
	if len(configs) == 0 {
		n.mu.Unlock()
		return false
	}

	// Snapshot configs to avoid racing with Create/Delete modifying the slice
	snapshot := make([]NotificationConfig, len(configs))
	copy(snapshot, configs)

	if n.lastSent == nil {
		n.lastSent = make(map[string]map[string]time.Time)
	}
	if timings, ok := n.lastSent[userID]; ok {
		if last, ok := timings[event.Type]; ok && time.Since(last) < n.rateLimit {
			n.mu.Unlock()
			return false
		}
	}
	if n.lastSent[userID] == nil {
		n.lastSent[userID] = make(map[string]time.Time)
	}
	n.lastSent[userID][event.Type] = time.Now()
	n.mu.Unlock()

	sent := false
	for _, cfg := range snapshot {
		if !cfg.Channel.Enabled {
			continue
		}
		if !n.ruleEnabled(cfg.Rules, event.Type) {
			continue
		}

		var err error
		switch cfg.Channel.Type {
		case "telegram":
			token, decodeErr := n.decryptToken(cfg.Channel.TokenEnc)
			if decodeErr != nil {
				log.Printf("notif: decrypt telegram token for %s: %v", userID, decodeErr)
				continue
			}
			err = sendTelegram(n.client, token, cfg.Channel.ChatID, event.Title, event.Message)
		case "slack":
			webhookURL, decodeErr := n.decryptToken(cfg.Channel.WebhookURL)
			if decodeErr != nil {
				log.Printf("notif: decrypt slack webhook for %s: %v", userID, decodeErr)
				continue
			}
			err = sendSlack(n.client, webhookURL, event.Title, event.Message)
		}
		if err != nil {
			log.Printf("notif: send to %s/%s failed: %v", userID, cfg.Channel.Type, err)
		} else {
			sent = true
		}
	}

	return sent
}

func (n *Notifier) ruleEnabled(rules NotificationRule, eventType string) bool {
	switch eventType {
	case "trade":
		return rules.TradeEvents
	case "order":
		return rules.OrderEvents
	case "container":
		return rules.ContainerHealth
	case "backtest":
		return true
	case "test":
		return true
	}
	return false
}

func (n *Notifier) decryptToken(enc string) (string, error) {
	if enc == "" {
		return "", fmt.Errorf("empty token")
	}
	return n.crypto.Decrypt(enc)
}

func (n *Notifier) EncryptToken(plain string) (string, error) {
	return n.crypto.Encrypt(plain)
}

func sendTelegram(client *http.Client, token, chatID, title, message string) error {
	text := fmt.Sprintf("*%s*\n%s", title, message)
	body := map[string]string{"chat_id": chatID, "text": text, "parse_mode": "Markdown"}
	data, _ := json.Marshal(body)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("telegram POST: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned %d", resp.StatusCode)
	}
	return nil
}

func sendSlack(client *http.Client, webhookURL, title, message string) error {
	body := map[string]string{"text": fmt.Sprintf("*%s*\n%s", title, message)}
	data, _ := json.Marshal(body)

	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("slack POST: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
