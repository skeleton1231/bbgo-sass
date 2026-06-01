package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateAndStart_FullDockerCommand_LiveMode verifies the complete docker
// run command for a live trading bot with real credentials.
func TestCreateAndStart_FullDockerCommand_LiveMode(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	creds := NewCredentialStore(dir, enc)
	insertTestCredential(t, creds, "test-user", "binance", "live-key", "live-secret")

	cfg := &Config{
		ManagerToken:   "tok",
		DataDir:        dir,
		DataVolume:     "bbgo-data",
		DockerNetwork:  "bbgo-net",
		BBGOImage:      "bbgo-base:latest",
		BBGOPort:       8080,
		BBGOGRPCPort:   9090,
		MarketDataAddr: "marketdata:9090",
	}
	cm := NewContainerManager(cfg, creds, nil)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	// Write YAML to disk so CreateAndStart can read it
	writeTestUserYAML(t, dir, "test-user", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2", Mode: "live",
			Config: rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`)},
	})

	if err := cm.CreateAndStart("test-user", ModeLive); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if !strings.Contains(cmdStr, "run -d") {
		t.Error("expected 'run -d'")
	}
	if !strings.Contains(cmdStr, "--name bbgo-test-user") {
		t.Error("expected --name bbgo-test-user")
	}
	if !strings.Contains(cmdStr, "--network bbgo-net") {
		t.Error("expected --network bbgo-net")
	}
	if !strings.Contains(cmdStr, "-v bbgo-data:/data") {
		t.Error("expected -v bbgo-data:/data")
	}
	if !strings.Contains(cmdStr, "--restart unless-stopped") {
		t.Error("expected --restart unless-stopped")
	}
	if strings.Contains(cmdStr, "PAPER_TRADE=1") {
		t.Error("PAPER_TRADE should NOT be set for live mode")
	}
	if !strings.Contains(cmdStr, "BINANCE_API_KEY=live-key") {
		t.Error("expected BINANCE_API_KEY injection")
	}
	if !strings.Contains(cmdStr, "BINANCE_API_SECRET=live-secret") {
		t.Error("expected BINANCE_API_SECRET injection")
	}
	if !strings.Contains(cmdStr, "DB_DRIVER=supabase") {
		t.Error("expected DB_DRIVER=supabase")
	}
	if !strings.Contains(cmdStr, "MARKET_DATA_SERVICE_URL=marketdata:9090") {
		t.Error("expected MARKET_DATA_SERVICE_URL injection")
	}
	if !strings.Contains(cmdStr, "bbgo-base:latest run") {
		t.Error("expected bbgo-base:latest run")
	}
	if !strings.Contains(cmdStr, "--config bbgo.yaml") {
		t.Error("expected --config bbgo.yaml")
	}
	if !strings.Contains(cmdStr, "--no-sync") {
		t.Error("expected --no-sync")
	}
	if !strings.Contains(cmdStr, "--enable-webserver") {
		t.Error("expected --enable-webserver")
	}
	if !strings.Contains(cmdStr, "--webserver-bind :8080") {
		t.Error("expected --webserver-bind :8080")
	}
	if !strings.Contains(cmdStr, "--enable-grpc") {
		t.Error("expected --enable-grpc")
	}
	if !strings.Contains(cmdStr, "--grpc-bind :9090") {
		t.Error("expected --grpc-bind :9090")
	}
}

// TestCreateAndStart_FullDockerCommand_PaperMode verifies paper mode has
// PAPER_TRADE=1 and no credential injection.
func TestCreateAndStart_FullDockerCommand_PaperMode(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
		BBGOGRPCPort:  9090,
	}
	cm := NewContainerManager(cfg, nil, nil)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	writeTestUserYAML(t, dir, "test-user", ModePaper, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2", Mode: "paper",
			Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
	})

	if err := cm.CreateAndStart("test-user", ModePaper); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if !strings.Contains(cmdStr, "-e PAPER_TRADE=1") {
		t.Error("expected PAPER_TRADE=1 for paper mode")
	}
	if strings.Contains(cmdStr, "BINANCE_API_KEY") {
		t.Error("no API keys should be injected without credentials")
	}
	if strings.Contains(cmdStr, "MARKET_DATA_SERVICE_URL") {
		t.Error("MARKET_DATA_SERVICE_URL should not be set when not configured")
	}
}

// TestCreateAndStart_YAMLWrittenForLive verifies the YAML file written to disk
// for live mode does not contain PAPER_TRADE and has correct strategy config.
func TestCreateAndStart_YAMLWrittenForLive(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerToken:  "tok",
		DataDir:       dir,
		DataVolume:    "bbgo-data",
		DockerNetwork: "bbgo-net",
		BBGOImage:     "bbgo-base:latest",
		BBGOPort:      8080,
	}
	cm := NewContainerManager(cfg, nil, nil)
	cm.dockerFn = func(args ...string) (string, error) {
		return "container-id", nil
	}

	writeTestUserYAML(t, dir, "test-user", ModeLive, []StrategyEntry{
		{Exchange: "binance", Strategy: "grid2", Mode: "live",
			Config: rawJSON(`{"symbol":"ETHUSDT","gridNumber":5}`)},
	})

	if err := cm.CreateAndStart("test-user", ModeLive); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	data, err := os.ReadFile(dir + "/test-user/bbgo.yaml")
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	yaml := string(data)

	if strings.Contains(yaml, "PAPER_TRADE") {
		t.Error("PAPER_TRADE should not appear in live mode YAML")
	}
	if !strings.Contains(yaml, "ETHUSDT") {
		t.Error("expected ETHUSDT symbol in YAML")
	}
	if !strings.Contains(yaml, "grid2:") {
		t.Error("expected grid2 strategy in YAML")
	}
	if !strings.Contains(yaml, "gridNumber:") {
		t.Error("expected gridNumber config in YAML")
	}
}

// writeTestUserYAML writes a bbgo.yaml for the given user/mode/strategies
// into the data directory structure expected by CreateAndStart.
func writeTestUserYAML(t *testing.T, dataDir, userID, mode string, strategies []StrategyEntry) {
	t.Helper()
	userDir := filepath.Join(dataDir, userID)
	if mode == ModePaper {
		userDir += "-paper"
	}
	os.MkdirAll(userDir, 0o755)

	yaml, err := buildUserYAML(userID, mode, strategies, func(string) bool { return false })
	if err != nil {
		t.Fatalf("build yaml for %s/%s: %v", userID, mode, err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "bbgo.yaml"), yaml, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}
