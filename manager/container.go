package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	runBacktestFn    func(userID string, jobID string, yamlContent []byte) ([]byte, error)
	syncBacktestFn   func(userID, exchange, symbol, start, end string) (string, error)
	logsFn           func(containerName string) (string, error)
	apiURLFn         func(containerName string) string
	checkRunningFn   func(containerName string) (bool, error)
	dockerFn         func(args ...string) (string, error)
	listRunningFn    func(userID string) map[string]bool
	listAllRunningFn func() map[string]bool
	checkHealthFn    func(containerName string) (ContainerHealth, error)
	captureErrorFn   func(containerName string) string

	proxyEnvFileOnce sync.Once
	proxyEnvFilePath string
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

// removeContainerIfExists force-removes a container by name (stopped or
// running). Used before `docker run --name <fixed>` to recover from a prior
// run that exited without --rm cleanup or hung (e.g. an infinite exchange-REST
// retry behind the GFW), which would otherwise leave the name in use and block
// every subsequent sync with a name conflict.
func (cm *ContainerManager) removeContainerIfExists(name string) {
	out, err := cm.docker("rm", "-f", name)
	if err != nil && !strings.Contains(out, "No such container") {
		log.Printf("remove stale container %s: %v (%s)", name, err, out)
	}
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
	// Instance dir must be writable by the bbgo container user (uid 10001), which
	// mkdirs {hostDir}/persistence/... at runtime. The manager runs as root, so
	// without this chown every strategy that uses persistence crashes on startup
	// with "mkdir .../persistence: permission denied". Backtest paths already do
	// this; live/paper instances were missing it.
	cm.chownToBBGO(hostDir)

	yamlPath := filepath.Join(hostDir, "bbgo.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		// Self-heal orphan instances: the Supabase row + config exist but the
		// on-disk bbgo.yaml was lost (volume cleanup, partially failed create,
		// etc.). Regenerate from the stored config so both start and the
		// container_recovery loop recover instead of failing forever on
		// "bbgo.yaml not found".
		log.Printf("instance %s missing bbgo.yaml, regenerating from stored config", inst.InstanceID)
		hasCredFn := func(exchange string) bool {
			if cm.creds == nil {
				return false
			}
			_, _, _, e := cm.creds.GetDecryptedByMode(inst.UserID, exchange, false)
			return e == nil
		}
		if yerr := cm.store.writeInstanceYAML(inst, hasCredFn); yerr != nil {
			return fmt.Errorf("regenerate bbgo.yaml for instance %s: %w", inst.InstanceID, yerr)
		}
	}

	containerDir := ContainerDir(inst.UserID, inst.Mode, inst.InstanceID)
	args := []string{
		"run", "-d",
		"--name", name,
		"--network", cm.cfg.DockerNetwork,
		"--network-alias", name,
		// Make host.docker.internal reachable so the propagated proxy URL
		// (collectedProxyEnv rewrites 127.0.0.1 -> host.docker.internal) resolves.
		// No-op on Docker Desktop (already resolved); required on Linux hosts.
		"--add-host", "host.docker.internal:host-gateway",
		"-v", cm.cfg.DataVolume + ":/data",
		"--workdir", containerDir,
		"--restart", "unless-stopped",
	}
	args = append(args, cm.instanceResourceArgs()...)
	args = append(args, cm.instanceEnvArgs(inst)...)
	args = append(args, cm.proxyEnvArgs()...)
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

// StopInstanceNames stops and removes the named containers in two batched
// docker calls (stop + rm) instead of 2N sequential spawns. Per-name fallback
// on batch failure so partial errors still get logged. Used by StopUser and
// ClearAllStrategies to avoid 2N docker CLI spawns over N instances.
func (cm *ContainerManager) StopInstanceNames(names []string) error {
	if len(names) == 0 {
		return nil
	}
	var firstErr error
	stopArgs := append([]string{"stop", "-t", "10"}, names...)
	if _, err := cm.docker(stopArgs...); err != nil {
		firstErr = fmt.Errorf("docker stop batch: %w", err)
		for _, name := range names {
			if _, err := cm.docker("stop", name, "-t", "10"); err != nil {
				log.Printf("stop %s: %v", name, err)
			}
		}
	}
	rmArgs := append([]string{"rm", "-f"}, names...)
	if _, err := cm.docker(rmArgs...); err != nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("docker rm batch: %w", err)
		}
		for _, name := range names {
			if _, err := cm.docker("rm", "-f", name); err != nil {
				log.Printf("rm %s: %v", name, err)
			}
		}
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

// ListRunningInstanceContainers returns the set of currently-running container
// names for the given user. One docker ps call instead of N per-instance
// docker inspect calls — used by ListBots to avoid an N+1 over the docker CLI.
func (cm *ContainerManager) ListRunningInstanceContainers(userID string) map[string]bool {
	if cm.listRunningFn != nil {
		return cm.listRunningFn(userID)
	}
	shortUser := userID
	if len(shortUser) > 8 {
		shortUser = shortUser[:8]
	}
	return cm.listRunningByPrefix("bbgo-" + shortUser + "-")
}

// ListAllRunningInstanceContainers returns all currently-running bbgo container
// names across all users. One docker ps call regardless of user/instance count.
// Used by Health for global stats — single call vs N×M docker inspect spawns.
func (cm *ContainerManager) ListAllRunningInstanceContainers() map[string]bool {
	if cm.listAllRunningFn != nil {
		return cm.listAllRunningFn()
	}
	return cm.listRunningByPrefix("bbgo-")
}

func (cm *ContainerManager) listRunningByPrefix(prefix string) map[string]bool {
	out, err := cm.docker("ps", "--filter", "name=^"+prefix, "--format", "{{.Names}}")
	if err != nil {
		return map[string]bool{}
	}
	set := map[string]bool{}
	for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
		name = strings.TrimSpace(name)
		if name != "" {
			set[name] = true
		}
	}
	return set
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

// rewriteLoopbackHost rewrites 127.0.0.1/localhost in the HOST portion of a URL
// to host.docker.internal so bridge-mode containers can reach the host's proxy.
// Uses url.Parse so paths/queries containing the literal "127.0.0.1" are not
// touched. Returns the input unchanged if parsing fails or the URL has no host.
func rewriteLoopbackHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	host := u.Hostname()
	if host == "127.0.0.1" || host == "localhost" {
		u.Host = strings.Replace(u.Host, host, "host.docker.internal", 1)
		return u.String()
	}
	return rawURL
}

// collectedProxyEnv returns KEY=VAL lines for the manager's proxy env vars,
// with loopback hosts rewritten to host.docker.internal. Empty if no proxy
// env is configured. NO_PROXY passes through unchanged.
func collectedProxyEnv() []string {
	var lines []string
	for _, k := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
		"http_proxy", "https_proxy", "all_proxy",
		"NO_PROXY", "no_proxy",
	} {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		if !strings.EqualFold(k, "NO_PROXY") {
			v = rewriteLoopbackHost(v)
		}
		lines = append(lines, k+"="+v)
	}
	return lines
}

// proxyEnvArgs returns docker args that propagate the manager's proxy env vars
// into spawned containers. Prefers a mode-0600 --env-file so proxy URLs (which
// may contain embedded credentials) are not exposed via docker inspect / ps /
// process accounting logs. Falls back to -e KEY=VAL args if env-file creation
// fails. Returns nil when no proxy env is configured.
//
// The env-file is created lazily on first call and reused for the ContainerManager's
// lifetime. Proxy env is read once; restart the manager to pick up changes.
func (cm *ContainerManager) proxyEnvArgs() []string {
	cm.proxyEnvFileOnce.Do(func() {
		lines := collectedProxyEnv()
		if len(lines) == 0 {
			return
		}
		dir := ""
		if cm.cfg != nil {
			dir = cm.cfg.DataDir
		}
		f, err := os.CreateTemp(dir, ".bbgo-proxy-env-*")
		if err != nil {
			log.Printf("proxy env file: %v — using -e args (proxy URLs visible via docker inspect)", err)
			return
		}
		for _, line := range lines {
			if _, err := fmt.Fprintln(f, line); err != nil {
				log.Printf("proxy env file write: %v — using -e args", err)
				f.Close()
				os.Remove(f.Name())
				return
			}
		}
		f.Chmod(0o600)
		f.Close()
		cm.proxyEnvFilePath = f.Name()
	})
	if cm.proxyEnvFilePath != "" {
		return []string{"--env-file", cm.proxyEnvFilePath}
	}
	var args []string
	for _, line := range collectedProxyEnv() {
		args = append(args, "-e", line)
	}
	return args
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
	args = append(args, "-e", "BBGO_HEARTBEAT_FILE="+filepath.Join(ContainerDir(inst.UserID, inst.Mode, inst.InstanceID), "heartbeat"))

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
	args = append(args, inst.RiskConfig.EnvArgs()...)

	args = append(args, cm.proxyEnvArgs()...)

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

// backtestSharedDirName is the directory (relative to DataDir) shared by sync
// and backtest containers for the kline cache DB (KLINE_DB_PATH) and the sync
// DB (DB_DSN). It must match the hardcoded "/data/backtest-shared" used in the
// docker -e flags below.
const backtestSharedDirName = "backtest-shared"

// ensureBacktestSharedDir creates the shared backtest directory used by sync and
// backtest containers. The directory is never created by bbgo itself (SQLite does
// not mkdir its parent), so the manager must create it up front.
func (cm *ContainerManager) ensureBacktestSharedDir() error {
	dir := filepath.Join(cm.cfg.DataDir, backtestSharedDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create backtest shared dir: %w", err)
	}
	cm.chownToBBGO(dir)
	return nil
}

// chownToBBGO best-effort transfers ownership of path to the bbgo container
// user (UID/GID from config, default 10001). The manager runs as root and
// creates directories root-owned, but bbgo containers run non-root and cannot
// write SQLite databases into root-owned directories — which surfaces as
// "unable to open database file". No-op when UID is 0 or the manager is not root.
func (cm *ContainerManager) chownToBBGO(path string) {
	if cm.cfg.BBGOUID == 0 && cm.cfg.BBGOGID == 0 {
		return
	}
	if err := os.Chown(path, cm.cfg.BBGOUID, cm.cfg.BBGOGID); err != nil {
		log.Printf("warning: chown %s to %d:%d failed: %v", path, cm.cfg.BBGOUID, cm.cfg.BBGOGID, err)
	}
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
	cm.chownToBBGO(hostDir)

	configPath := filepath.Join(hostDir, "bbgo.yaml")
	if err := os.WriteFile(configPath, yamlContent, 0o644); err != nil {
		return nil, fmt.Errorf("write backtest config: %w", err)
	}

	if err := cm.ensureBacktestSharedDir(); err != nil {
		return nil, err
	}

	cDir := backtestContainerDir(userID, jobID)
	args := []string{
		"run", "--rm",
		"--name", "bt-" + jobID,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		// Make host.docker.internal resolve so the propagated proxy env
		// (proxyEnvArgs) works — lets bbgo fetch klines directly from the
		// exchange when marketdata lacks them (e.g. 1m behind the GFW).
		"--add-host", "host.docker.internal:host-gateway",
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
	args = append(args, cm.proxyEnvArgs()...)
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

	if err := cm.ensureBacktestSharedDir(); err != nil {
		return "", err
	}

	cDir := backtestContainerDir(userID, syncID)
	args := []string{
		"run", "--rm",
		"--name", "bt-" + syncID,
		"--network", cm.cfg.DockerNetwork,
		"-v", cm.cfg.DataVolume + ":/data",
		"--add-host", "host.docker.internal:host-gateway",
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
	args = append(args, cm.proxyEnvArgs()...)

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

	// bt-<syncID> is a fixed name reused for every sync of this exchange;
	// clear any stale/hung leftover so it does not block this run with a
	// name conflict.
	cm.removeContainerIfExists("bt-" + syncID)

	out, err := cm.dockerLong(args...)
	if err != nil {
		return out, fmt.Errorf("sync failed: %s: %w", out, err)
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
