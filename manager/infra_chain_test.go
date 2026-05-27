package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExchangeEnvPrefix_AllKnownExchanges(t *testing.T) {
	expected := map[string]string{
		"binance":  "BINANCE",
		"okex":     "OKEX",
		"kucoin":   "KUCOIN",
		"bybit":    "BYBIT",
		"bitget":   "BITGET",
		"max":      "MAX",
		"coinbase": "COINBASE",
		"bitfinex": "BITFINEX",
	}
	for ex, want := range expected {
		got := exchangeEnvPrefix(ex)
		if got != want {
			t.Errorf("exchangeEnvPrefix(%q) = %q, want %q", ex, got, want)
		}
	}
}

func TestExchangeEnvPrefix_UnknownFallsBack(t *testing.T) {
	got := exchangeEnvPrefix("unknown_exchange")
	if got != "EXCHANGE" {
		t.Errorf("exchangeEnvPrefix(unknown) = %q, want EXCHANGE", got)
	}
}

func TestContainerName_Format(t *testing.T) {
	cm := &ContainerManager{}
	name := cm.containerName("abc-123", ModeLive)
	if name != "bbgo-abc-123" {
		t.Errorf("containerName = %q, want bbgo-abc-123", name)
	}
}

func TestHostDir_And_UserDir(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{DataDir: "/data"}}
	if got := cm.hostDir("u1", ModeLive); got != "/data/u1" {
		t.Errorf("hostDir = %q, want /data/u1", got)
	}
	if got := cm.userDir("u1", ModeLive); got != "/data/u1" {
		t.Errorf("userDir = %q, want /data/u1", got)
	}
}

func TestAPIURL_UsesDockerDNS(t *testing.T) {
	cm := &ContainerManager{cfg: &Config{BBGOPort: 8080}}
	url := cm.APIURL("user42", ModeLive)
	if url != "http://bbgo-user42:8080" {
		t.Errorf("APIURL = %q, want http://bbgo-user42:8080", url)
	}
}

func TestAPIURL_TestHook(t *testing.T) {
	cm := &ContainerManager{
		cfg:      &Config{BBGOPort: 8080},
		apiURLFn: func(userID, mode string) string { return "http://mock:9999" },
	}
	if got := cm.APIURL("any", ModeLive); got != "http://mock:9999" {
		t.Errorf("APIURL with hook = %q, want http://mock:9999", got)
	}
}

func TestCleanupBackups_KeepsNewest(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"bbgo.db.backup.2026-01-01-000000",
		"bbgo.db.backup.2026-02-01-000000",
		"bbgo.db.backup.2026-03-01-000000",
		"bbgo.db.backup.2026-04-01-000000",
		"bbgo.db.backup.2026-05-01-000000",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cleanupBackups(dir, "bbgo.db.backup", 3)

	entries, _ := os.ReadDir(dir)
	remaining := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bbgo.db.backup.") {
			remaining++
		}
	}
	if remaining != 3 {
		t.Errorf("expected 3 remaining backups, got %d", remaining)
	}

	data, err := os.ReadFile(filepath.Join(dir, "bbgo.db.backup.2026-05-01-000000"))
	if err != nil {
		t.Fatal("newest backup should survive")
	}
	if string(data) != "x" {
		t.Error("backup content corrupted")
	}
}

func TestCleanupBackups_NothingToDelete(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"bbgo.db.backup.2026-05-01-000000",
		"bbgo.db.backup.2026-05-02-000000",
	} {
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644)
	}
	cleanupBackups(dir, "bbgo.db.backup", 3)

	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("expected 2 files untouched, got %d", len(entries))
	}
}

func TestBuildSyncConfig_YAMLStructure(t *testing.T) {
	yamlBytes, err := buildSyncConfig("binance", "BTCUSDT", "2025-01-01", "2025-06-01")
	if err != nil {
		t.Fatalf("buildSyncConfig: %v", err)
	}
	s := string(yamlBytes)

	if !strings.Contains(s, "binance:") {
		t.Error("expected binance session")
	}
	if !strings.Contains(s, "BTCUSDT") {
		t.Error("expected BTCUSDT symbol")
	}
	if !strings.Contains(s, "2025-01-01") {
		t.Error("expected startTime")
	}
	if !strings.Contains(s, "2025-06-01") {
		t.Error("expected endTime")
	}
}

func TestNormalizeStrategyConfig_LegacyAlias(t *testing.T) {
	s, p := normalizeStrategyConfig("sentinel_anomaly", map[string]interface{}{"threshold": 0.5})
	if s != "sentinel" {
		t.Errorf("got %q, want sentinel", s)
	}
	if p["threshold"] != 0.5 {
		t.Error("params should be preserved")
	}
}

func TestNormalizeStrategyConfig_FieldAlias(t *testing.T) {
	params := map[string]interface{}{"interval": "1h", "symbol": "BTCUSDT"}
	s, p := normalizeStrategyConfig("dca", params)
	if s != "dca" {
		t.Errorf("got %q, want dca", s)
	}
	if _, hasOld := p["interval"]; hasOld {
		t.Error("old field 'interval' should be removed")
	}
	if p["investmentInterval"] != "1h" {
		t.Error("expected investmentInterval renamed from interval")
	}
	if p["symbol"] != "BTCUSDT" {
		t.Error("unrelated fields should be preserved")
	}
}

func TestBuildUserYAML_LegacyAlias_Normalized(t *testing.T) {
	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "u1",
		Strategies: []StrategyEntry{
			{Strategy: "sentinel_anomaly", Exchange: "binance", Mode: "live",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	yaml, err := buildUserYAML(uc, func(ex string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	s := string(yaml)
	if strings.Contains(s, "sentinel_anomaly") {
		t.Error("legacy alias should be normalized to sentinel")
	}
	if !strings.Contains(s, "sentinel:") {
		t.Error("expected sentinel strategy in YAML")
	}
}

func TestCloneUserContainer_DeepCopy(t *testing.T) {
	original := &UserContainer{
		Mode:   ModeLive,
		UserID: "u1",
		Status: StatusRunning,
		Strategies: []StrategyEntry{
			{ID: "s1", Strategy: "grid2", Exchange: "binance"},
		},
	}
	cloned := cloneUserContainer(original)

	cloned.Strategies[0].ID = "modified"
	if original.Strategies[0].ID != "s1" {
		t.Error("clone should not share slice backing with original")
	}
}

func TestDBBackup_OnCreateAndStart(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "test-user")
	os.MkdirAll(userDir, 0o755)

	dbPath := filepath.Join(userDir, "bbgo.db")
	os.WriteFile(dbPath, []byte("fake-db-content"), 0o644)

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

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart: %v", err)
	}

	// DB must still exist (not renamed away)
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("bbgo.db should be preserved: %v", err)
	}

	entries, _ := os.ReadDir(userDir)
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bbgo.db.backup.") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected backup file bbgo.db.backup.* to exist")
	}
}

func TestDBBackup_NoDB_NoError(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "test-user")
	os.MkdirAll(userDir, 0o755)

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

	uc := &UserContainer{
		Mode:   ModeLive,
		UserID: "test-user",
		Strategies: []StrategyEntry{
			{Exchange: "binance", Strategy: "grid2", Mode: "paper",
				Config: rawJSON(`{"symbol":"BTCUSDT"}`)},
		},
	}
	if err := cm.CreateAndStart(uc); err != nil {
		t.Fatalf("CreateAndStart without prior db: %v", err)
	}
}
