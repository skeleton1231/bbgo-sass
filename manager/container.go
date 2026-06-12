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
	store *InstanceStore

	// test hooks
	runBacktestFn  func(userID string, jobID string, yamlContent []byte) ([]byte, error)
	syncBacktestFn func(userID, exchange, symbol, start, end string) (string, error)
	logsFn         func(containerName string) (string, error)
	apiURLFn       func(containerName string) string
	checkRunningFn func(containerName string) (bool, error)
	dockerFn       func(args ...string) (string, error)
}

func NewContainerManager(cfg *Config, creds *CredentialStore, p *pool.Pool, store *InstanceStore) *ContainerManager {
	return &ContainerManager{cfg: cfg, creds: creds, pool: p, store: store}
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

func (cm *ContainerManager) InstanceContainerName(userID, mode, instanceID string) string {
	return instanceSlug(userID, mode, instanceID)
}

func (cm *ContainerManager) InstanceAPIURL(userID, mode, instanceID string) string {
	name := cm.InstanceContainerName(userID, mode, instanceID)
	if cm.apiURLFn != nil {
		return cm.apiURLFn(name)
	}
	return fmt.Sprintf("http://%s:%d", name, cm.cfg.BBGOPort)
}

func (cm *ContainerManager) CreateAndStartInstance(inst *StrategyInstance) error {
	name := cm.InstanceContainerName(inst.UserID, inst.Mode, inst.InstanceID)
	_ = cm.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)

	hostDir := cm.store.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	yamlPath := filepath.Join(hostDir, "bbgo.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		return fmt.Errorf("bbgo.yaml not found for instance %s: %w", inst.InstanceID, err)
	}

	containerDir := ContainerDir(inst.UserID, inst.Mode, inst.InstanceID)
	args := []string{
		"run", "-d",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"--network-alias", name,
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", containerDir,
		"--restart", "unless-stopped",
	}
	args = append(args, cm.instanceResourceArgs()...)
	args = append(args, cm.instanceEnvArgs(inst)...)
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

	log.Printf("instance container %s started (image: %s)", name, cm.cfg.BBGOImage)
	return nil
}

// StopInstance stops and removes the container. Returns an error from docker so
// callers that gate subsequent work on a clean stop (e.g., restart-after-edit)
// can surface failures instead of reporting success on a still-running container.
// Best-effort callers may discard the error.
func (cm *ContainerManager) StopInstance(userID, mode, instanceID string) error {
	name := cm.InstanceContainerName(userID, mode, instanceID)
	var firstErr error
	if _, err := cm.docker("stop", name, "-t", "10"); err != nil {
		firstErr = fmt.Errorf("docker stop %s: %w", name, err)
	}
	if _, err := cm.docker("rm", "-f", name); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("docker rm %s: %w", name, err)
	}
	return firstErr
}

func (cm *ContainerManager) IsInstanceRunning(userID, mode, instanceID string) bool {
	running, _ := cm.CheckInstanceRunning(userID, mode, instanceID)
	return running
}

func (cm *ContainerManager) CheckInstanceRunning(userID, mode, instanceID string) (bool, error) {
	name := cm.InstanceContainerName(userID, mode, instanceID)
	if cm.checkRunningFn != nil {
		return cm.checkRunningFn(name)
	}
	out, err := cm.docker("inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, err
	}
	return out == "true", nil
}

func (cm *ContainerManager) InstanceLogs(userID, mode, instanceID, tail string) (string, error) {
	if cm.logsFn != nil {
		return cm.logsFn(cm.InstanceContainerName(userID, mode, instanceID))
	}
	name := cm.InstanceContainerName(userID, mode, instanceID)
	out, err := cm.docker("logs", "--tail", tail, name)
	if err != nil {
		return "", fmt.Errorf("docker logs %s: %w", name, err)
	}
	return out, nil
}

func (cm *ContainerManager) InstanceGRPCAddr(userID, mode, instanceID string) string {
	return fmt.Sprintf("%s:%d", cm.InstanceContainerName(userID, mode, instanceID), cm.cfg.BBGOGRPCPort)
}

func (cm *ContainerManager) instanceEnvArgs(inst *StrategyInstance) []string {
	args := []string{}

	if inst.Mode == ModePaper {
		args = append(args, "-e", "PAPER_TRADE=1")
	}

	args = append(args, "-e", "BBGO_STRATEGY_INSTANCE_ID="+inst.InstanceID)

	args = append(args,
		"-e", "DB_DRIVER=postgresql",
		"-e", "SUPABASE_DB_URL="+cm.cfg.SupabaseDBURL,
		"-e", "BBGO_USER_ID="+inst.UserID,
	)

	if inst.Mode == ModePaper {
		args = append(args, "-e", "SUPABASE_TABLE_PREFIX=paper_")
	}
	args = append(args, "-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db")

	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}

	if cm.creds != nil && inst.Mode != ModePaper {
		exchanges := []string{inst.Exchange}
		if inst.CrossExchange {
			exchanges = nil
			for _, sr := range inst.Sessions {
				exchanges = append(exchanges, sr.Exchange)
			}
		}
		injected := map[string]bool{}
		for _, ex := range exchanges {
			if injected[ex] {
				continue
			}
			apiKey, apiSecret, passphrase, err := cm.creds.GetDecryptedByMode(inst.UserID, ex, false)
			if err == nil {
				prefix := exchangeEnvPrefix(ex)
				args = append(args,
					"-e", prefix+"_API_KEY="+apiKey,
					"-e", prefix+"_API_SECRET="+apiSecret,
				)
				if passphrase != "" {
					args = append(args, "-e", prefix+"_API_PASSPHRASE="+passphrase)
				}
			}
			injected[ex] = true
		}
	}

	return args
}

func (cm *ContainerManager) instanceResourceArgs() []string {
	r := cm.cfg.InstanceResources
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

func (cm *ContainerManager) FindRunningInstance(userID string) (*StrategyInstance, error) {
	if cm.store == nil {
		return nil, fmt.Errorf("no running container found, please start a trading container first")
	}
	instances, err := cm.store.ListAllInstances(userID)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		inst := &instances[i]
		if cm.IsInstanceRunning(inst.UserID, inst.Mode, inst.InstanceID) {
			return inst, nil
		}
	}
	return nil, fmt.Errorf("no running container found, please start a trading container first")
}

// --- Backtest ---

func (cm *ContainerManager) backtestDir(userID, jobID string) string {
	return filepath.Join(cm.cfg.DataDir, "backtest", userID, jobID)
}

func backtestContainerDir(userID, jobID string) string {
	return fmt.Sprintf("/data/backtest/%s/%s", userID, jobID)
}

func (cm *ContainerManager) backtestResourceArgs() []string {
	r := cm.cfg.BacktestResources
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
	return args
}

func (cm *ContainerManager) RunBacktest(userID string, jobID string, yamlContent []byte) ([]byte, error) {
	if cm.runBacktestFn != nil {
		return cm.runBacktestFn(userID, jobID, yamlContent)
	}

	if strings.Contains(jobID, "/") || strings.Contains(jobID, "..") {
		return nil, fmt.Errorf("invalid job ID: %s", jobID)
	}

	hostDir := cm.backtestDir(userID, jobID)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backtest dir: %w", err)
	}

	configPath := filepath.Join(hostDir, "bbgo.yaml")
	if err := os.WriteFile(configPath, yamlContent, 0o644); err != nil {
		return nil, fmt.Errorf("write backtest config: %w", err)
	}

	cDir := backtestContainerDir(userID, jobID)
	args := []string{
		"run", "--rm",
		"--name", "bt-" + jobID,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
	}
	args = append(args, cm.backtestResourceArgs()...)
	args = append(args,
		"-e", "DB_DRIVER=sqlite3",
		"-e", "DB_DSN="+cDir+"/bbgo.db",
		"-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db",
		"-e", "BINANCE_TESTNET=0",
		"-e", "PAPER_TRADE=0",
	)
	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}
	args = append(args,
		cm.cfg.BBGOImage,
		"backtest",
		"--sync",
		"--config", cDir+"/bbgo.yaml",
		"--output", cDir,
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
	dir := cm.backtestDir(userID, jobID)
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("cleanup backtest %s for user %s: %v", jobID, userID, err)
	}
}

func (cm *ContainerManager) BacktestReportDir(userID, jobID string) string {
	return cm.backtestDir(userID, jobID)
}

func (cm *ContainerManager) ReadBacktestReport(userID, jobID string) (json.RawMessage, []byte, error) {
	reportDir := cm.BacktestReportDir(userID, jobID)

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

	yamlBytes, err := buildSyncConfig(exchange, symbol, startTime, endTime)
	if err != nil {
		return "", err
	}

	syncID := "sync-" + exchange
	hostDir := cm.backtestDir(userID, syncID)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	configPath := filepath.Join(hostDir, "sync.yaml")
	if err := os.WriteFile(configPath, yamlBytes, 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	cDir := backtestContainerDir(userID, syncID)
	args := []string{
		"run", "--rm",
		"--name", "bt-" + syncID,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
	}
	args = append(args, cm.backtestResourceArgs()...)
	args = append(args,
		"-e", "KLINE_DB_PATH=/data/backtest-shared/backtest.db",
		"-e", "BINANCE_TESTNET=0",
		"-e", "PAPER_TRADE=0",
		"-e", "DB_DRIVER=sqlite3",
		"-e", "DB_DSN=/data/backtest-shared/sync.db",
	)
	if cm.cfg.MarketDataAddr != "" {
		args = append(args, "-e", "MARKET_DATA_SERVICE_URL="+cm.cfg.MarketDataAddr)
	}

	syncFrom := startTime
	if t, err := time.Parse(time.RFC3339, startTime); err == nil {
		syncFrom = t.Format("2006-01-02")
	}

	args = append(args,
		cm.cfg.BBGOImage,
		"backtest",
		"--sync",
		"--sync-only",
		"--sync-from", syncFrom,
		"--config", cDir+"/sync.yaml",
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

func (cm *ContainerManager) DiscoverContainers() []StrategyInstance {
	out, err := cm.docker("ps", "--filter", "name="+containerPrefix, "--format", "{{.Names}}")
	if err != nil {
		log.Printf("discover containers: %v", err)
		return nil
	}

	var instances []StrategyInstance
	for _, name := range strings.Split(out, "\n") {
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, containerPrefix) {
			continue
		}
		suffix := strings.TrimPrefix(name, containerPrefix)
		parts := strings.SplitN(suffix, "-", 3)
		if len(parts) < 3 {
			continue
		}
		userID := parts[0]
		mode := parts[1]
		if mode != ModeLive && mode != ModePaper {
			continue
		}
		instanceID := parts[2]
		if userID == "" || userID == "marketdata" {
			continue
		}
		instances = append(instances, StrategyInstance{
			InstanceID: instanceID,
			UserID:     userID,
			Mode:       mode,
		})
	}
	return instances
}

func (cm *ContainerManager) StopAllForUser(userID string) {
	instances, err := cm.store.ListAllInstances(userID)
	if err != nil {
		return
	}
	for _, inst := range instances {
		_ = cm.StopInstance(inst.UserID, inst.Mode, inst.InstanceID)
	}
}

func (cm *ContainerManager) StartAllForUser(userID, mode string) []error {
	instances, err := cm.store.ListInstances(userID, mode)
	if err != nil {
		return []error{err}
	}
	var errs []error
	for i := range instances {
		if err := cm.CreateAndStartInstance(&instances[i]); err != nil {
			errs = append(errs, fmt.Errorf("instance %s: %w", instances[i].InstanceID, err))
		}
	}
	return errs
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
