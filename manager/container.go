package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
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

func (cm *ContainerManager) CreateAndStart(uc *UserContainer) error {
	name := cm.containerName(uc.UserID)
	cm.StopAndRemove(uc.UserID)

	dir := cm.userDir(uc.UserID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	dbPath := dir + "/bbgo.db"
	if _, err := os.Stat(dbPath); err == nil {
		backup := dbPath + ".backup." + time.Now().Format("20060102-150405")
		os.Rename(dbPath, backup)
		log.Printf("backed up %s → %s", dbPath, backup)
	}

	yamlContent := buildUserYAML(uc)
	if err := os.WriteFile(dir+"/bbgo.yaml", []byte(yamlContent), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", dir,
		"--restart", "unless-stopped",
	}
	args = append(args, cm.envArgs(uc)...)
	args = append(args,
		cm.cfg.BBGOImage,
		"run",
		"--config", "bbgo.yaml",
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
	dir := cm.userDir(uc.UserID)
	yamlContent := buildUserYAML(uc)
	if err := os.WriteFile(dir+"/bbgo.yaml", []byte(yamlContent), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	name := cm.containerName(uc.UserID)
	out, err := cm.docker("restart", name)
	if err != nil {
		return fmt.Errorf("docker restart: %s: %w", out, err)
	}
	log.Printf("container %s restarted", name)
	return nil
}

func (cm *ContainerManager) RunBacktest(userID string, yamlContent []byte) ([]byte, error) {
	tmpDir := fmt.Sprintf("/data/%s/backtest", userID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backtest dir: %w", err)
	}

	configPath := fmt.Sprintf("%s/bbgo.yaml", tmpDir)
	if err := os.WriteFile(configPath, yamlContent, 0o644); err != nil {
		return nil, fmt.Errorf("write backtest config: %w", err)
	}

	name := fmt.Sprintf("bbgo-backtest-%d", os.Getpid())
	args := []string{
		"run", "--rm",
		"--name", name,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", tmpDir,
		"-e", "DB_DRIVER=sqlite3",
		"-e", fmt.Sprintf("DB_DSN=%s/backtest.db", tmpDir),
		cm.cfg.BBGOImage,
		"backtest",
		"--config", "bbgo.yaml",
	}

	out, err := cm.docker(args...)
	if err != nil {
		return nil, fmt.Errorf("backtest failed: %s: %w", out, err)
	}
	return []byte(out), nil
}

func (cm *ContainerManager) envArgs(uc *UserContainer) []string {
	args := []string{}
	if len(uc.Strategies) > 0 && uc.Strategies[0].Mode == "paper" {
		args = append(args, "-e", "PAPER_TRADE=1")
	}

	dir := cm.userDir(uc.UserID)
	args = append(args,
		"-e", "DB_DRIVER=sqlite3",
		"-e", fmt.Sprintf("DB_DSN=%s/bbgo.db", dir),
	)

	if cm.creds != nil && len(uc.Strategies) > 0 {
		exchange := uc.Strategies[0].Exchange
		if apiKey, apiSecret, passphrase, err := cm.creds.GetDecrypted(uc.UserID, exchange); err == nil {
			prefix := exchangeEnvPrefix(exchange)
			args = append(args,
				"-e", prefix+"_API_KEY="+apiKey,
				"-e", prefix+"_API_SECRET="+apiSecret,
			)
			if passphrase != "" {
				args = append(args, "-e", prefix+"_PASSPHRASE="+passphrase)
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
	name := cm.containerName(userID)
	out, _ := cm.docker("inspect", "-f", "{{.State.Running}}", name)
	return out == "true"
}

func (cm *ContainerManager) APIURL(userID string) string {
	name := cm.containerName(userID)
	return fmt.Sprintf("http://%s:%d", name, cm.cfg.BBGOPort)
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
