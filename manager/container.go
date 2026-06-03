package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/c9s/bbgo/saas/manager/pool"
	"gopkg.in/yaml.v3"
)

const dockerTimeout = 2 * time.Minute

const backtestDockerTimeout = 30 * time.Minute

const containerPrefix = "bbgo-"

type ContainerManager struct {
	cfg   *Config
	creds *CredentialStore
	pool  *pool.Pool

	// test hooks
	runBacktestFn  func(userID string, jobID string, yamlContent []byte) ([]byte, error)
	syncBacktestFn func(userID, exchange, symbol, start, end string) (string, error)
	logsFn         func(userID, mode string) (string, error)
	apiURLFn       func(userID, mode string) string
	checkRunningFn func(userID, mode string) (bool, error)
	dockerFn       func(args ...string) (string, error)
}

func NewContainerManager(cfg *Config, creds *CredentialStore, p *pool.Pool) *ContainerManager {
	return &ContainerManager{cfg: cfg, creds: creds, pool: p}
}

func (cm *ContainerManager) containerName(userID, mode string) string {
	name := containerPrefix + userID
	if mode == ModePaper {
		name += "-paper"
	}
	return name
}

func (cm *ContainerManager) docker(args ...string) (string, error) {
	if cm.dockerFn != nil {
		return cm.dockerFn(args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), dockerTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (cm *ContainerManager) dockerLong(args ...string) (string, error) {
	if cm.dockerFn != nil {
		return cm.dockerFn(args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), backtestDockerTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
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

func (cm *ContainerManager) userDir(userID, mode string) string {
	dir := fmt.Sprintf("/data/%s", userID)
	if mode == ModePaper {
		dir += "-paper"
	}
	return dir
}

func (cm *ContainerManager) hostDir(userID, mode string) string {
	dir := cm.cfg.DataDir + "/" + userID
	if mode == ModePaper {
		dir += "-paper"
	}
	return dir
}

// APIURL returns the internal Docker DNS URL for the user's bbgo container.
func (cm *ContainerManager) APIURL(userID, mode string) string {
	if cm.apiURLFn != nil {
		return cm.apiURLFn(userID, mode)
	}
	return fmt.Sprintf("http://%s:%d", cm.containerName(userID, mode), cm.cfg.BBGOPort)
}

func (cm *ContainerManager) CreateAndStart(userID, mode string) error {
	name := cm.containerName(userID, mode)
	cm.StopAndRemove(userID, mode)

	hDir := cm.hostDir(userID, mode)
	if err := os.MkdirAll(hDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// bbgo.yaml should already exist on disk (written by StrategyStore)
	yamlPath := hDir + "/bbgo.yaml"
	if _, err := os.Stat(yamlPath); err != nil {
		return fmt.Errorf("bbgo.yaml not found for user %s: %w", userID, err)
	}

	containerDir := cm.userDir(userID, mode)
	strategies, _ := parseStrategiesFromYAMLFile(yamlPath)
	args := []string{
		"run", "-d",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", containerDir,
		"--restart", "unless-stopped",
	}
	args = append(args, cm.resourceArgs(mode)...)
	args = append(args, cm.envArgs(userID, mode, strategies)...)
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

func (cm *ContainerManager) Restart(userID, mode string) error {
	return cm.CreateAndStart(userID, mode)
}

func parseStrategiesFromYAMLFile(path string) ([]StrategyEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseStrategiesFromYAML(data)
}

// backtestMode returns the mode of an available container for backtest operations.
// It prefers the paper container but falls back to the live container.
func (cm *ContainerManager) backtestMode(userID string) (string, error) {
	if cm.IsRunning(userID, ModePaper) {
		return ModePaper, nil
	}
	if cm.IsRunning(userID, ModeLive) {
		return ModeLive, nil
	}
	return "", fmt.Errorf("no running container found, please start a trading container first")
}

func (cm *ContainerManager) RunBacktest(userID string, jobID string, yamlContent []byte) ([]byte, error) {
	if cm.runBacktestFn != nil {
		return cm.runBacktestFn(userID, jobID, yamlContent)
	}

	if strings.Contains(jobID, "/") || strings.Contains(jobID, "..") {
		return nil, fmt.Errorf("invalid job ID: %s", jobID)
	}

	mode, err := cm.backtestMode(userID)
	if err != nil {
		return nil, err
	}

	hostBacktestDir := cm.hostDir(userID, mode) + "/backtest/" + jobID
	if err := os.MkdirAll(hostBacktestDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backtest dir: %w", err)
	}

	configPath := hostBacktestDir + "/bbgo.yaml"
	if err := os.WriteFile(configPath, yamlContent, 0o644); err != nil {
		return nil, fmt.Errorf("write backtest config: %w", err)
	}

	containerName := cm.containerName(userID, mode)
	containerBacktestDir := cm.userDir(userID, mode) + "/backtest/" + jobID
	containerConfigPath := containerBacktestDir + "/bbgo.yaml"
	containerDbPath := containerBacktestDir + "/bbgo.db"
	args := []string{
		"exec",
		"-e", "DB_DRIVER=sqlite3",
		"-e", "DB_DSN=" + containerDbPath,
		"-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db",
		"-e", "BINANCE_TESTNET=0",
		"-e", "PAPER_TRADE=0",
	}
	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}
	args = append(args,
		containerName,
		"bbgo", "backtest",
		"--sync",
		"--config", containerConfigPath,
		"--output", containerBacktestDir,
	)

	out, err := cm.dockerLong(args...)
	if err != nil {
		return nil, fmt.Errorf("backtest failed: %s: %w", out, err)
	}
	return []byte(out), nil
}

func (cm *ContainerManager) CleanupBacktest(userID, jobID string) {
	if cm == nil {
		return
	}
	mode, err := cm.backtestMode(userID)
	if err != nil {
		return
	}
	dir := cm.hostDir(userID, mode) + "/backtest/" + jobID
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("cleanup backtest %s for user %s: %v", jobID, userID, err)
	}
}

func (cm *ContainerManager) BacktestReportDir(userID, jobID string) string {
	mode, err := cm.backtestMode(userID)
	if err != nil {
		return ""
	}
	return cm.hostDir(userID, mode) + "/backtest/" + jobID
}

func (cm *ContainerManager) ReadBacktestReport(userID, jobID string) (json.RawMessage, []byte, error) {
	reportDir := cm.BacktestReportDir(userID, jobID)
	if reportDir == "" {
		return nil, nil, fmt.Errorf("cannot resolve report dir for %s", jobID)
	}

	summaryPath := filepath.Join(reportDir, "summary.json")
	summaryData, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read summary.json: %w", err)
	}

	var equityCurve []byte
	equityPath := filepath.Join(reportDir, "equity_curve.tsv")
	if data, err := os.ReadFile(equityPath); err == nil {
		equityCurve = data
	}

	return json.RawMessage(summaryData), equityCurve, nil
}

func (cm *ContainerManager) SyncBacktest(userID, exchange, symbol, startTime, endTime string) (string, error) {
	if cm.syncBacktestFn != nil {
		return cm.syncBacktestFn(userID, exchange, symbol, startTime, endTime)
	}

	mode, err := cm.backtestMode(userID)
	if err != nil {
		return "", err
	}

	yamlBytes, err := buildSyncConfig(exchange, symbol, startTime, endTime)
	if err != nil {
		return "", err
	}

	hostBacktestDir := cm.hostDir(userID, mode) + "/backtest"
	if err := os.MkdirAll(hostBacktestDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	configPath := hostBacktestDir + "/sync.yaml"
	if err := os.WriteFile(configPath, yamlBytes, 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	containerName := cm.containerName(userID, mode)
	containerConfigPath := cm.userDir(userID, mode) + "/backtest/sync.yaml"
	args := []string{
		"exec",
		"-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db",
		"-e", "BINANCE_TESTNET=0",
		"-e", "PAPER_TRADE=0",
	}
	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}

	syncFrom := startTime
	if t, err := time.Parse(time.RFC3339, startTime); err == nil {
		syncFrom = t.Format("2006-01-02")
	}

	args = append(args,
		containerName,
		"bbgo", "backtest",
		"--sync",
		"--sync-only",
		"--sync-from", syncFrom,
		"--config", containerConfigPath,
	)

	out, err := cm.dockerLong(args...)
	if err != nil {
		return out, fmt.Errorf("sync failed: %w", err)
	}
	return out, nil
}

func buildSyncConfig(exchange, symbol, startTime, endTime string) ([]byte, error) {
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

	return yaml.Marshal(&cfg)
}

func (cm *ContainerManager) resourceArgs(mode string) []string {
	r := cm.cfg.ResourcesForMode(mode)
	var args []string
	if r.Memory != "" {
		args = append(args, "--memory", r.Memory)
	}
	if r.MemorySwap != "" {
		args = append(args, "--memory-swap", r.MemorySwap)
	}
	if r.CPUs != "" {
		args = append(args, "--cpus", r.CPUs)
	}
	if r.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", r.PidsLimit))
	}
	if r.LogMaxSize != "" {
		args = append(args, "--log-opt", "max-size="+r.LogMaxSize)
	}
	if r.LogMaxFile > 0 {
		args = append(args, "--log-opt", fmt.Sprintf("max-file=%d", r.LogMaxFile))
	}
	return args
}

func (cm *ContainerManager) envArgs(userID, mode string, strategies []StrategyEntry) []string {
	args := []string{}

	// Paper containers get PAPER_TRADE=1 (bbgo uses this to enable testnet mode)
	if mode == ModePaper {
		args = append(args, "-e", "PAPER_TRADE=1")
	}

	// Use "/" not filepath.Join — this path runs inside the Linux container.
	if mode == ModePaper {
		args = append(args,
			"-e", "DB_DRIVER=sqlite3",
			"-e", "DB_DSN="+cm.userDir(userID, ModePaper)+"/bbgo.db",
		)
	} else {
		args = append(args,
			"-e", "DB_DRIVER=supabase",
			"-e", "SUPABASE_URL="+cm.cfg.SupabaseURL,
			"-e", "SUPABASE_SERVICE_KEY="+cm.cfg.SupabaseKey,
			"-e", "BBGO_USER_ID="+userID,
		)
	}
	args = append(args, "-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db")

	if cm.cfg.MarketDataAddr != "" {
		addr := cm.cfg.MarketDataAddr
		if mode == ModePaper && cm.cfg.MarketDataTestnetAddr != "" {
			addr = cm.cfg.MarketDataTestnetAddr
		}
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+addr)
	}

	// Inject credentials: paper mode only injects Binance testnet creds
	if cm.creds != nil {
		injected := map[string]bool{}
		wantTestnet := mode == ModePaper
		for _, s := range strategies {
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
				// Paper mode: only inject Binance testnet credentials
				if mode == ModePaper && ex != paperExchange {
					injected[ex] = true
					continue
				}
				apiKey, apiSecret, passphrase, err := cm.creds.GetDecryptedByMode(userID, ex, wantTestnet)
				if err == nil {
					prefix := exchangeEnvPrefix(ex)
					args = append(args,
						"-e", prefix+"_API_KEY="+apiKey,
						"-e", prefix+"_API_SECRET="+apiSecret,
					)
					if passphrase != "" {
						args = append(args, "-e", prefix+"_API_PASSPHRASE="+passphrase)
					}
					if mode == ModePaper {
						args = append(args, "-e", prefix+"_TESTNET=1")
					}
				}
				injected[ex] = true
			}
		}
	}

	return args
}

func (cm *ContainerManager) Stop(userID, mode string) {
	name := cm.containerName(userID, mode)
	out, err := cm.docker("inspect", "-f", "{{.State.Running}}", name)
	if err != nil || out != "true" {
		return
	}
	cm.docker("stop", name, "-t", "10")
	log.Printf("container %s stopped", name)
}

func (cm *ContainerManager) StopAndRemove(userID, mode string) {
	name := cm.containerName(userID, mode)
	cm.docker("stop", name, "-t", "10")
	cm.docker("rm", "-f", name)
}

func (cm *ContainerManager) IsRunning(userID, mode string) bool {
	running, _ := cm.CheckRunning(userID, mode)
	return running
}

func (cm *ContainerManager) CheckRunning(userID, mode string) (bool, error) {
	if cm.checkRunningFn != nil {
		return cm.checkRunningFn(userID, mode)
	}
	name := cm.containerName(userID, mode)
	out, err := cm.docker("inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, err
	}
	return out == "true", nil
}

func (cm *ContainerManager) Logs(userID, mode, tail string) (string, error) {
	if cm.logsFn != nil {
		return cm.logsFn(userID, mode)
	}
	name := cm.containerName(userID, mode)
	out, err := cm.docker("logs", "--tail", tail, name)
	if err != nil {
		return "", fmt.Errorf("docker logs %s: %w", name, err)
	}
	return out, nil
}

// ContainerGRPCAddr returns the gRPC address for a user's container.
func (cm *ContainerManager) ContainerGRPCAddr(userID, mode string) string {
	return fmt.Sprintf("%s:%d", cm.containerName(userID, mode), cm.cfg.BBGOGRPCPort)
}


// DiscoverContainers scans Docker for running bbgo-* containers and returns
// the userIDs and modes found. Used during startup to detect orphaned containers
// that don't have corresponding bbgo.yaml configs on disk.
func (cm *ContainerManager) DiscoverContainers() map[string][]string {
	out, err := cm.docker("ps", "--filter", "name="+containerPrefix, "--format", "{{.Names}}")
	if err != nil {
		log.Printf("discover containers: %v", err)
		return nil
	}

	result := make(map[string][]string)
	for _, name := range strings.Split(out, "\n") {
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, containerPrefix) {
			continue
		}
		suffix := strings.TrimPrefix(name, containerPrefix)

		var userID, mode string
		if strings.HasSuffix(suffix, "-paper") {
			userID = strings.TrimSuffix(suffix, "-paper")
			mode = ModePaper
		} else {
			userID = suffix
			mode = ModeLive
		}
		if userID == "" || userID == "marketdata" {
			continue
		}
		result[userID] = append(result[userID], mode)
	}
	return result
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := out.ReadFrom(in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	return nil
}

// cleanupBackups keeps only the keepNewest most recent backup files matching
// the given prefix pattern in dir. Older backups are deleted.
func cleanupBackups(dir, prefix string, keepNewest int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var backups []os.DirEntry
	for _, e := range entries {
		if matched, _ := filepath.Match(prefix+"*", e.Name()); matched {
			backups = append(backups, e)
		}
	}
	if len(backups) <= keepNewest {
		return
	}
	sort.Slice(backups, func(i, j int) bool {
		ni, errI := backups[i].Info()
		nj, errJ := backups[j].Info()
		if errI != nil || errJ != nil {
			return errI == nil
		}
		return ni.ModTime().After(nj.ModTime())
	})
	for _, b := range backups[keepNewest:] {
		os.Remove(filepath.Join(dir, b.Name()))
	}
}
