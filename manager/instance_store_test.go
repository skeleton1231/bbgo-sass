package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewInstanceStore(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	if store == nil || store.dataDir != dir {
		t.Error("NewInstanceStore failed")
	}
}

func TestInstanceStore_SetSupabase(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.SetSupabase(nil)
	if store.sb != nil {
		t.Error("should be nil")
	}
}

func TestInstanceStore_Registry_Nil(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	if store.Registry() != nil {
		t.Error("expected nil registry")
	}
}

func TestInstanceStore_Registry_NonNil(t *testing.T) {
	dir := t.TempDir()
	reg := &StrategyDefaultsCache{}
	store := NewInstanceStore(dir, reg)
	if store.Registry() != reg {
		t.Error("registry mismatch")
	}
}

func TestInstanceStore_Defaults_Nil(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	if store.Defaults() != nil {
		t.Error("expected nil defaults")
	}
}

func TestInstanceStore_Defaults_NonNil(t *testing.T) {
	dir := t.TempDir()
	reg := &StrategyDefaultsCache{}
	store := NewInstanceStore(dir, reg)
	if store.Defaults() == nil {
		t.Error("expected non-nil defaults")
	}
}

func TestInstanceStore_IsLiveOnly_NilRegistry(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	if store.IsLiveOnly("grid2") {
		t.Error("nil registry should not be liveOnly")
	}
}

func TestInstanceStore_InstanceDir(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	result := store.InstanceDir("user1", "live", "grid2-btcusdt")
	if result != filepath.Join(dir, "user1", "live", "grid2-btcusdt") {
		t.Errorf("InstanceDir = %q", result)
	}
}

func TestInstanceStore_YAMLExists_False(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	if store.YAMLExists("user1", "live", "inst1") {
		t.Error("should not exist")
	}
}

func TestInstanceStore_YAMLExists_True(t *testing.T) {
	store, _ := newTestStore(t)
	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	if !store.YAMLExists(testUUID, "live", inst.InstanceID) {
		t.Error("should exist after creation")
	}
}

func TestInstanceStore_CreateInstance(t *testing.T) {
	store, _ := newTestStore(t)
	inst := &StrategyInstance{
		UserID: testUUID, Mode: "live", Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT", Config: rawJSON(`{}`),
	}
	inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, inst.Config)
	if err := store.CreateInstance(inst, func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if !store.YAMLExists(testUUID, "live", inst.InstanceID) {
		t.Error("yaml should exist after creation")
	}
}

func TestInstanceStore_CreateInstance_Duplicate(t *testing.T) {
	store, _ := newTestStore(t)
	inst := &StrategyInstance{
		UserID: testUUID, Mode: "live", Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT", Config: rawJSON(`{}`),
	}
	inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, inst.Config)
	store.CreateInstance(inst, func(string) bool { return false })
	err := store.CreateInstance(inst, func(string) bool { return false })
	if err == nil {
		t.Error("expected error for duplicate instance")
	}
}

func TestInstanceStore_RemoveInstance(t *testing.T) {
	store, _ := newTestStore(t)
	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	if err := store.RemoveInstance(testUUID, "live", inst.InstanceID); err != nil {
		t.Fatal(err)
	}
	if store.YAMLExists(testUUID, "live", inst.InstanceID) {
		t.Error("should not exist after removal")
	}
}

func TestInstanceStore_GetInstance(t *testing.T) {
	store, _ := newTestStore(t)
	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	got, err := store.GetInstance(testUUID, "live", inst.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Strategy != "grid2" {
		t.Errorf("strategy = %q", got.Strategy)
	}
}

func TestInstanceStore_GetInstance_EmptyMode(t *testing.T) {
	store, _ := newTestStore(t)
	inst := createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	got, err := store.GetInstance(testUUID, "", inst.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Strategy != "grid2" {
		t.Errorf("strategy = %q", got.Strategy)
	}
}

func TestInstanceStore_GetInstance_NotFound(t *testing.T) {
	store, _ := newTestStore(t)
	_, err := store.GetInstance(testUUID, "live", "nonexistent")
	if err == nil {
		t.Error("expected error for missing instance")
	}
}

func TestInstanceStore_ListInstances(t *testing.T) {
	store, _ := newTestStore(t)
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	createTestInstance(t, store, testUUID, "live", "supertrend", "ETHUSDT", nil)
	instances, err := store.ListInstances(testUUID, "live")
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 2 {
		t.Errorf("expected 2, got %d", len(instances))
	}
}

func TestInstanceStore_ListInstances_Empty(t *testing.T) {
	store, _ := newTestStore(t)
	instances, err := store.ListInstances(testUUID, "live")
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 0 {
		t.Errorf("expected 0, got %d", len(instances))
	}
}

func TestInstanceStore_ListAllInstances(t *testing.T) {
	store, _ := newTestStore(t)
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	createTestInstance(t, store, testUUID, "paper", "supertrend", "ETHUSDT", nil)
	all, err := store.ListAllInstances(testUUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestInstanceStore_ScanUsers(t *testing.T) {
	store, _ := newTestStore(t)
	createTestInstance(t, store, testUUID, "live", "grid2", "BTCUSDT", nil)
	users := store.ScanUsers()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].UserID != testUUID || users[0].Mode != "live" {
		t.Errorf("user = %+v", users[0])
	}
}

func TestInstanceStore_ScanUsers_Empty(t *testing.T) {
	store, _ := newTestStore(t)
	if users := store.ScanUsers(); len(users) != 0 {
		t.Errorf("expected 0, got %d", len(users))
	}
}

func TestExtractSymbolFromConfig(t *testing.T) {
	if s := extractSymbolFromConfig(rawJSON(`{"symbol":"BTCUSDT"}`)); s != "BTCUSDT" {
		t.Errorf("symbol = %q", s)
	}
	if s := extractSymbolFromConfig(rawJSON(`{}`)); s != "" {
		t.Errorf("empty = %q", s)
	}
	if s := extractSymbolFromConfig(nil); s != "" {
		t.Errorf("nil = %q", s)
	}
}

func TestParseExchangeStrategyEntry(t *testing.T) {
	entry, ok := parseExchangeStrategyEntry(map[string]any{
		"on": "binance", "grid2": map[string]any{"symbol": "BTCUSDT"},
	})
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Exchange != "binance" || entry.Strategy != "grid2" {
		t.Errorf("entry = %+v", entry)
	}
}

func TestParseExchangeStrategyEntry_NoOn(t *testing.T) {
	_, ok := parseExchangeStrategyEntry(map[string]any{"grid2": map[string]any{}})
	if ok {
		t.Error("should fail without 'on' key")
	}
}

func TestParseCrossStrategyEntry(t *testing.T) {
	sessions := map[string]sessionConfig{
		"maker": {Exchange: "binance", EnvVarPrefix: "BINANCE"},
		"hedge": {Exchange: "bybit", EnvVarPrefix: "BYBIT", Futures: true},
	}
	entry, ok := parseCrossStrategyEntry(map[string]any{
		"xmaker": map[string]any{"symbol": "BTCUSDT"},
	}, sessions)
	if !ok {
		t.Fatal("expected ok")
	}
	if entry.Strategy != "xmaker" || !entry.CrossExchange {
		t.Errorf("entry = %+v", entry)
	}
	if len(entry.Sessions) != 2 {
		t.Errorf("sessions = %d", len(entry.Sessions))
	}
}

func TestParseInstanceYAML(t *testing.T) {
	yamlContent := []byte(`
exchange:
  binance:
    symbol: BTCUSDT
sessions:
  binance:
    exchange: binance
    envVarPrefix: BINANCE
exchangeStrategies:
  - on: binance
    grid2:
      symbol: BTCUSDT
      gridNumber: 10
`)
	inst, err := parseInstanceYAML(yamlContent, testUUID, "live", "test-dir")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Strategy != "grid2" {
		t.Errorf("strategy = %q", inst.Strategy)
	}
	if inst.Exchange != "binance" {
		t.Errorf("exchange = %q", inst.Exchange)
	}
}

func TestParseInstanceYAML_CrossExchange(t *testing.T) {
	yamlContent := []byte(`
exchange:
  binance:
    symbol: BTCUSDT
  bybit:
    symbol: BTCUSDT
sessions:
  maker:
    exchange: binance
    envVarPrefix: BINANCE
  hedge:
    exchange: bybit
    envVarPrefix: BYBIT
crossExchangeStrategies:
  - xmaker:
      symbol: BTCUSDT
`)
	inst, err := parseInstanceYAML(yamlContent, testUUID, "live", "test-dir")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Strategy != "xmaker" {
		t.Errorf("strategy = %q", inst.Strategy)
	}
	if !inst.CrossExchange {
		t.Error("should be cross exchange")
	}
}

func TestParseInstanceYAML_InvalidYAML(t *testing.T) {
	_, err := parseInstanceYAML([]byte("invalid: [yaml: broken"), testUUID, "live", "test-dir")
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestBuildInstanceYAML_PaperMode(t *testing.T) {
	inst := &StrategyInstance{
		UserID: testUUID, Mode: ModePaper, Strategy: "grid2",
		Exchange: "binance", Symbol: "BTCUSDT",
		Config: rawJSON(`{"symbol":"BTCUSDT"}`),
	}
	inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, inst.Config)
	yamlBytes, err := buildInstanceYAML(inst, func(string) bool { return false }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(yamlBytes) == 0 {
		t.Error("expected non-empty YAML")
	}
}

func TestRowToInstance(t *testing.T) {
	r := PublicStrategyInstancesSelect{
		InstanceId: "inst1", UserId: "u1", Mode: "live",
		Strategy: "grid2", Exchange: "binance", Symbol: "BTCUSDT",
		Config: json.RawMessage(`{"symbol":"BTCUSDT"}`), Name: "test",
	}
	inst := rowToInstance(r)
	if inst.InstanceID != "inst1" || inst.Strategy != "grid2" {
		t.Errorf("inst = %+v", inst)
	}
}

func TestRowToInstance_WithSessions(t *testing.T) {
	r := PublicStrategyInstancesSelect{
		InstanceId: "inst1", UserId: "u1", Mode: "live",
		Strategy: "xmaker", Exchange: "binance", Symbol: "BTCUSDT",
		Sessions: json.RawMessage(`[{"name":"maker","exchange":"binance"}]`),
	}
	inst := rowToInstance(r)
	if len(inst.Sessions) != 1 || inst.Sessions[0].Name != "maker" {
		t.Errorf("sessions = %+v", inst.Sessions)
	}
}

func TestInstanceStore_dbPath(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	p := store.dbPath("u1", "live", "inst1")
	if p != filepath.Join(dir, "u1", "live", "inst1", "bbgo.db") {
		t.Errorf("dbPath = %q", p)
	}
}

func TestInstanceStore_listFromDisk_NoDir(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	instances, err := store.listFromDisk("nonexistent", "live")
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 0 {
		t.Errorf("expected empty, got %d", len(instances))
	}
}

func TestInstanceStore_getFromDisk_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	_, err := store.getFromDisk("u1", "live", "nonexistent")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestInstanceStore_upsertToSupabase_Nil(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.upsertToSupabase(&StrategyInstance{InstanceID: "inst1"})
}

func TestInstanceStore_deleteFromSupabase_Nil(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	store.deleteFromSupabase("u1", "live", "inst1")
}

func TestInstanceStore_RemoveInstance_NonexistentDir(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	err := store.RemoveInstance("nonexistent", "live", "inst1")
	if err != nil {
		t.Errorf("RemoveInstance on nonexistent should not error: %v", err)
	}
}

func TestInstanceStore_yamlPath(t *testing.T) {
	dir := t.TempDir()
	store := NewInstanceStore(dir, nil)
	p := store.yamlPath("u1", "live", "inst1")
	if p != filepath.Join(dir, "u1", "live", "inst1", "bbgo.yaml") {
		t.Errorf("yamlPath = %q", p)
	}
}

func TestInstanceStore_listFromDisk_BadYAML(t *testing.T) {
	dir := t.TempDir()
	modeDir := filepath.Join(dir, testUUID, "live", "bad-inst")
	os.MkdirAll(modeDir, 0o755)
	os.WriteFile(filepath.Join(modeDir, "bbgo.yaml"), []byte("not: valid: yaml: [broken"), 0o644)
	store := NewInstanceStore(dir, nil)
	instances, err := store.listFromDisk(testUUID, "live")
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 0 {
		t.Errorf("bad YAML should produce 0 instances, got %d", len(instances))
	}
}
