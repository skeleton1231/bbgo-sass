package main

import (
	"strings"
	"testing"
)

func TestCreateAndStartInstance_FullDockerCommand_LiveMode(t *testing.T) {
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
	store := NewInstanceStore(dir, nil)
	cm := NewContainerManager(cfg, creds, nil, store)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModeLive,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT","gridNumber":10}`),
		InstanceID: "grid2-BTCUSDT",
	}
	if err := store.CreateInstance(inst, func(string) bool { return true }); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("CreateAndStartInstance: %v", err)
	}

	cmdStr := strings.Join(capturedArgs, " ")

	if !strings.Contains(cmdStr, "run -d") {
		t.Error("expected 'run -d'")
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

func TestCreateAndStartInstance_FullDockerCommand_PaperMode(t *testing.T) {
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
	store := NewInstanceStore(dir, nil)
	cm := NewContainerManager(cfg, nil, nil, store)

	var capturedArgs []string
	cm.dockerFn = func(args ...string) (string, error) {
		capturedArgs = args
		return "container-id", nil
	}

	inst := &StrategyInstance{
		UserID:     "test-user",
		Mode:       ModePaper,
		Strategy:   "grid2",
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		Config:     rawJSON(`{"symbol":"BTCUSDT"}`),
		InstanceID: "grid2-BTCUSDT",
	}
	if err := store.CreateInstance(inst, func(string) bool { return false }); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := cm.CreateAndStartInstance(inst); err != nil {
		t.Fatalf("CreateAndStartInstance: %v", err)
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
