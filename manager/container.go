package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const containerPrefix = "bbgo-"

type ContainerManager struct {
	cfg   *Config
	creds *CredentialStore
}

func NewContainerManager(cfg *Config, creds *CredentialStore) *ContainerManager {
	return &ContainerManager{cfg: cfg, creds: creds}
}

func (cm *ContainerManager) containerName(userID string) string {
	return containerPrefix + userID
}

func (cm *ContainerManager) docker(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (cm *ContainerManager) EnsureNetwork() error {
	out, err := cm.docker("network", "create", cm.cfg.DockerNetwork)
	if err != nil && !strings.Contains(out, "already exists") {
		return fmt.Errorf("create network: %s: %w", out, err)
	}
	return nil
}

func (cm *ContainerManager) userDir(userID string) string {
	return fmt.Sprintf("/data/%s", userID)
}

func (cm *ContainerManager) hostDir(userID string) string {
	return cm.cfg.DataDir + "/" + userID
}

// APIURL returns the internal Docker DNS URL for the user's bbgo container.
// Works when manager and containers share the same Docker network.
func (cm *ContainerManager) APIURL(userID string) string {
	return fmt.Sprintf("http://%s:%d", cm.containerName(userID), cm.cfg.BBGOPort)
}

func (cm *ContainerManager) CreateAndStart(uc *UserContainer) error {
	name := cm.containerName(uc.UserID)
	cm.StopAndRemove(uc.UserID)

	hostDir := cm.hostDir(uc.UserID)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	dbPath := hostDir + "/bbgo.db"
	if _, err := os.Stat(dbPath); err == nil {
		backup := dbPath + ".backup." + time.Now().Format("20060102-150405")
		os.Rename(dbPath, backup)
		log.Printf("backed up %s -> %s", dbPath, backup)
	}

	yamlContent := buildUserYAML(uc, func(exchange string) bool {
		_, _, _, err := cm.creds.GetDecrypted(uc.UserID, exchange)
		return err == nil
	})
	if err := os.WriteFile(hostDir+"/bbgo.yaml", []byte(yamlContent), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	containerDir := cm.userDir(uc.UserID)
	args := []string{
		"run", "-d",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", containerDir,
		"--restart", "unless-stopped",
	}
	args = append(args, cm.envArgs(uc)...)
	args = append(args,
		cm.cfg.BBGOImage,
		"run",
		"--config", "bbgo.yaml",
		"--no-sync",
		"--enable-webserver",
		"--webserver-bind", fmt.Sprintf(":%d", cm.cfg.BBGOPort),
		"--enable-grpc",
		"--grpc-bind", fmt.Sprintf(":%d", cm.cfg.BBGOGRPCPort),
	)

	out, err := cm.docker(args...)
	if err != nil {
		return fmt.Errorf("docker run: %s: %w", out, err)
	}

	log.Printf("container %s started (image: %s)", name, cm.cfg.BBGOImage)
	return nil
}

func (cm *ContainerManager) Restart(uc *UserContainer) error {
	return cm.CreateAndStart(uc)
}

func (cm *ContainerManager) RunBacktest(userID string, yamlContent []byte) ([]byte, error) {
	hostBacktestDir := cm.hostDir(userID) + "/backtest"
	if err := os.MkdirAll(hostBacktestDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backtest dir: %w", err)
	}

	cm.ensureBacktestSharedDir()

	configPath := hostBacktestDir + "/bbgo.yaml"
	if err := os.WriteFile(configPath, yamlContent, 0o644); err != nil {
		return nil, fmt.Errorf("write backtest config: %w", err)
	}

	containerDir := cm.userDir(userID) + "/backtest"
	name := fmt.Sprintf("bbgo-bt-%s-%d", safeShortID(userID), time.Now().UnixNano())
	args := []string{
		"run", "--rm",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", containerDir,
		"-e", "DB_DRIVER=sqlite3",
		"-e", "DB_DSN=/data/backtest-shared/backtest.db",
	}
	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}
	args = append(args,
		cm.cfg.BBGOImage,
		"backtest",
		"--sync",
		"--config", "bbgo.yaml",
	)

	out, err := cm.docker(args...)
	if err != nil {
		return nil, fmt.Errorf("backtest failed: %s: %w", out, err)
	}
	return []byte(out), nil
}

func (cm *ContainerManager) SyncBacktest(exchange, symbol, startTime, endTime string) (string, error) {
	cm.ensureBacktestSharedDir()

	type syncConfig struct {
		Sessions map[string]struct {
			Exchange string `yaml:"exchange"`
		} `yaml:"sessions"`
		Backtest struct {
			Sessions  []string `yaml:"sessions"`
			Symbols   []string `yaml:"symbols"`
			StartTime string   `yaml:"startTime"`
			EndTime   string   `yaml:"endTime"`
		} `yaml:"backtest"`
	}

	cfg := syncConfig{}
	cfg.Sessions = map[string]struct {
		Exchange string `yaml:"exchange"`
	}{
		exchange: {Exchange: exchange},
	}
	cfg.Backtest.Sessions = []string{exchange}
	cfg.Backtest.Symbols = []string{symbol}
	cfg.Backtest.StartTime = startTime
	cfg.Backtest.EndTime = endTime

	yamlBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", fmt.Errorf("marshal backtest config: %w", err)
	}
	yamlContent := string(yamlBytes)

	hostDir := cm.cfg.DataDir + "/backtest-sync"
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	configPath := hostDir + "/bbgo.yaml"
	if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	name := fmt.Sprintf("bbgo-sync-%d", time.Now().UnixNano())
	args := []string{
		"run", "--rm",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", "/data/backtest-sync",
		"-e", "DB_DRIVER=sqlite3",
		"-e", "DB_DSN=/data/backtest-shared/backtest.db",
	}
	args = append(args,
		cm.cfg.BBGOImage,
		"backtest",
		"--sync",
		"--sync-only",
		"--sync-from", startTime,
		"--config", "bbgo.yaml",
	)

	out, err := cm.docker(args...)
	if err != nil {
		return out, fmt.Errorf("sync failed: %w", err)
	}
	return out, nil
}

func (cm *ContainerManager) ensureBacktestSharedDir() {
	args := []string{
		"run", "--rm",
		"-v", cm.cfg.DataVolume + ":/data",
		"--entrypoint", "sh",
		cm.cfg.BBGOImage,
		"-c", "mkdir -p /data/backtest-shared",
	}
	if _, err := cm.docker(args...); err != nil {
		log.Printf("backtest-shared dir ensure (may already exist): %v", err)
	}
}

func (cm *ContainerManager) envArgs(uc *UserContainer) []string {
	args := []string{}
	for _, s := range uc.Strategies {
		if s.Mode == "paper" {
			args = append(args, "-e", "PAPER_TRADE=1")
			break
		}
	}

	dir := cm.userDir(uc.UserID)
	args = append(args,
		"-e", "DB_DRIVER=sqlite3",
		"-e", fmt.Sprintf("DB_DSN=%s/bbgo.db", dir),
		"-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db",
	)

	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}

	if cm.creds != nil {
		injected := map[string]bool{}
		for _, s := range uc.Strategies {
			exchanges := []string{}
			if s.CrossExchange {
				for _, sr := range s.Sessions {
					exchanges = append(exchanges, sr.Exchange)
				}
			} else if s.Exchange != "" {
				exchanges = append(exchanges, s.Exchange)
			}
			for _, ex := range exchanges {
				if injected[ex] {
					continue
				}
				if apiKey, apiSecret, passphrase, err := cm.creds.GetDecrypted(uc.UserID, ex); err == nil {
					prefix := exchangeEnvPrefix(ex)
					args = append(args,
						"-e", prefix+"_API_KEY="+apiKey,
						"-e", prefix+"_API_SECRET="+apiSecret,
					)
					if passphrase != "" {
						args = append(args, "-e", prefix+"_PASSPHRASE="+passphrase)
					}
					injected[ex] = true
				}
			}
		}
	}

	return args
}

func (cm *ContainerManager) Stop(userID string) {
	name := cm.containerName(userID)
	out, err := cm.docker("inspect", "-f", "{{.State.Running}}", name)
	if err != nil || out != "true" {
		return
	}
	cm.docker("stop", name, "-t", "10")
	log.Printf("container %s stopped", name)
}

func (cm *ContainerManager) StopAndRemove(userID string) {
	name := cm.containerName(userID)
	cm.docker("stop", name, "-t", "10")
	cm.docker("rm", "-f", name)
}

func (cm *ContainerManager) IsRunning(userID string) bool {
	running, _ := cm.CheckRunning(userID)
	return running
}

func (cm *ContainerManager) CheckRunning(userID string) (bool, error) {
	name := cm.containerName(userID)
	out, err := cm.docker("inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, err
	}
	return out == "true", nil
}

func (cm *ContainerManager) Logs(userID string, tail string) (string, error) {
	name := cm.containerName(userID)
	out, err := cm.docker("logs", "--tail", tail, name)
	if err != nil {
		return "", fmt.Errorf("docker logs %s: %w", name, err)
	}
	return out, nil
}

func (cm *ContainerManager) RecoverUsers(users []*UserContainer) {
	for _, uc := range users {
		name := cm.containerName(uc.UserID)
		out, _ := cm.docker("inspect", "-f", "{{.State.Running}}", name)
		if out == "true" {
			log.Printf("recovered container %s (running)", name)
			uc.Status = StatusRunning
		} else if uc.Status == StatusRunning {
			log.Printf("recovering container %s for user %s", name, uc.UserID)
			if err := cm.CreateAndStart(uc); err != nil {
				log.Printf("recover user %s failed: %v", uc.UserID, err)
			}
		}
	}
}

var exchangePrefixes = map[string]string{
	"binance":  "BINANCE",
	"okex":     "OKEX",
	"kucoin":   "KUCOIN",
	"bybit":    "BYBIT",
	"bitget":   "BITGET",
	"max":      "MAX",
	"coinbase": "COINBASE",
	"bitfinex": "BITFINEX",
}

func exchangeEnvPrefix(exchange string) string {
	if p, ok := exchangePrefixes[exchange]; ok {
		return p
	}
	return "EXCHANGE"
}
