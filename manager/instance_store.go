package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

type InstanceStore struct {
	mu       sync.Mutex
	dataDir  string
	registry *StrategyDefaultsCache
	sb       *SupabaseClient

	supabaseUpsertFn func(inst *StrategyInstance)
}

func NewInstanceStore(dataDir string, registry *StrategyDefaultsCache) *InstanceStore {
	return &InstanceStore{dataDir: dataDir, registry: registry}
}

func (s *InstanceStore) SetSupabase(sb *SupabaseClient) {
	s.sb = sb
}

func (s *InstanceStore) Registry() *StrategyDefaultsCache {
	return s.registry
}

func (s *InstanceStore) IsLiveOnly(strategyID string) bool {
	if s.registry == nil {
		return false
	}
	return s.registry.IsLiveOnly(strategyID)
}

func (s *InstanceStore) Defaults() DefaultsProvider {
	if s.registry == nil {
		return nil
	}
	return s.registry
}

// InstanceDir returns the host filesystem path for an instance's data directory.
func (s *InstanceStore) InstanceDir(userID, mode, instanceID string) string {
	return filepath.Join(s.dataDir, userID, mode, instanceDirName(instanceID))
}

func (s *InstanceStore) yamlPath(userID, mode, instanceID string) string {
	return filepath.Join(s.InstanceDir(userID, mode, instanceID), "bbgo.yaml")
}

func (s *InstanceStore) dbPath(userID, mode, instanceID string) string {
	return filepath.Join(s.InstanceDir(userID, mode, instanceID), "bbgo.db")
}

// ContainerDir returns the in-container path for an instance's data directory.
func ContainerDir(userID, mode, instanceID string) string {
	return fmt.Sprintf("/data/%s/%s/%s", userID, mode, instanceDirName(instanceID))
}

// CreateInstance writes bbgo.yaml, creates the directory, and persists to Supabase.
func (s *InstanceStore) CreateInstance(inst *StrategyInstance, hasCredentials func(string) bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.InstanceDir(inst.UserID, inst.Mode, inst.InstanceID)
	if s.YAMLExists(inst.UserID, inst.Mode, inst.InstanceID) {
		return fmt.Errorf("instance %s already exists", inst.InstanceID)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	yamlContent, err := buildInstanceYAML(inst, hasCredentials, s.registry)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.yamlPath(inst.UserID, inst.Mode, inst.InstanceID), yamlContent, 0o644); err != nil {
		return err
	}

	s.upsertToSupabase(inst)
	return nil
}

// UpdateInstanceFuturesConfig updates the FuturesConfig on an existing instance using
// merge semantics (zero-valued fields in fc do NOT clear existing values), regenerates
// its bbgo.yaml on disk, and upserts the row to Supabase.
// Caller is responsible for restarting the container if it is currently running.
func (s *InstanceStore) UpdateInstanceFuturesConfig(userID, mode, instanceID string, fc *FuturesConfig, hasCredentials func(string) bool) (*StrategyInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, err := s.getFromDisk(userID, mode, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance %s not found: %w", instanceID, err)
	}
	inst.FuturesConfig = mergeFuturesConfig(inst.FuturesConfig, fc)

	if err := s.writeInstanceYAML(inst, hasCredentials); err != nil {
		return nil, err
	}
	return inst, nil
}

// writeInstanceYAML regenerates bbgo.yaml from inst and upserts the row to
// Supabase. Callers hold s.mu and have already mutated inst in place.
func (s *InstanceStore) writeInstanceYAML(inst *StrategyInstance, hasCredentials func(string) bool) error {
	yamlContent, err := buildInstanceYAML(inst, hasCredentials, s.registry)
	if err != nil {
		return fmt.Errorf("rebuild yaml: %w", err)
	}
	if err := os.WriteFile(s.yamlPath(inst.UserID, inst.Mode, inst.InstanceID), yamlContent, 0o644); err != nil {
		return fmt.Errorf("write yaml: %w", err)
	}
	s.upsertToSupabase(inst)
	return nil
}

// mergeFuturesConfig applies PATCH-style merge semantics: zero-valued fields in
// `patch` do not overwrite the corresponding fields in `base`. Returns a new
// pointer; both inputs are left untouched. If patch is nil the result is base.
func mergeFuturesConfig(base, patch *FuturesConfig) *FuturesConfig {
	if patch == nil {
		return base
	}
	out := &FuturesConfig{}
	if base != nil {
		*out = *base
	}
	if patch.Leverage != 0 {
		out.Leverage = patch.Leverage
	}
	if patch.MarginType != "" {
		out.MarginType = patch.MarginType
	}
	return out
}

// UpdateInstanceRiskConfig updates the universal RiskConfig on an existing
// instance using merge semantics (zero-valued fields in patch do NOT clear
// existing values), regenerates its bbgo.yaml on disk, and upserts the row
// to Supabase. Caller is responsible for restarting the container if it is
// currently running.
func (s *InstanceStore) UpdateInstanceRiskConfig(userID, mode, instanceID string, rc *RiskConfig, hasCredentials func(string) bool) (*StrategyInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, err := s.getFromDisk(userID, mode, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance %s not found: %w", instanceID, err)
	}
	inst.RiskConfig = mergeRiskConfig(inst.RiskConfig, rc)

	if err := s.writeInstanceYAML(inst, hasCredentials); err != nil {
		return nil, err
	}
	return inst, nil
}

// mergeRiskConfig applies PATCH-style merge semantics: zero-valued fields in
// `patch` do NOT overwrite corresponding fields in `base`. Returns a new
// pointer; inputs are untouched. To clear a field, send 0 explicitly via
// the dedicated clear endpoint (TBD). If patch is nil the result is base.
// If both are nil/empty the result is nil (so EnvArgs returns nothing and
// bbgo's UniversalRiskController stays disabled).
func mergeRiskConfig(base, patch *RiskConfig) *RiskConfig {
	if patch == nil {
		return base
	}
	out := &RiskConfig{}
	if base != nil {
		*out = *base
	}
	if patch.StopLossPrice != 0 {
		out.StopLossPrice = patch.StopLossPrice
	}
	if patch.TakeProfitPrice != 0 {
		out.TakeProfitPrice = patch.TakeProfitPrice
	}
	if patch.RoiStopLoss != 0 {
		out.RoiStopLoss = patch.RoiStopLoss
	}
	if patch.RoiTakeProfit != 0 {
		out.RoiTakeProfit = patch.RoiTakeProfit
	}
	if patch.TrailingActivation != 0 {
		out.TrailingActivation = patch.TrailingActivation
	}
	if patch.TrailingCallback != 0 {
		out.TrailingCallback = patch.TrailingCallback
	}
	if patch.MaxPositionQty != 0 {
		out.MaxPositionQty = patch.MaxPositionQty
	}
	if !out.HasAny() {
		return nil
	}
	return out
}

// UpdateInstanceConfig deep-merges a patch over the instance's strategy params
// JSON, regenerates bbgo.yaml on disk, and upserts the row to Supabase.
// Caller is responsible for restarting the container if it is currently
// running.
//
// Symbol and strategy cannot be changed via this path: the deterministic
// instance ID is computed from those fields (see pkg/instanceid), so mutating
// them would orphan historical trades/orders under the old ID and break data
// isolation. Callers wanting a different symbol or strategy must delete and
// recreate the instance.
func (s *InstanceStore) UpdateInstanceConfig(userID, mode, instanceID string, patch map[string]any, hasCredentials func(string) bool) (*StrategyInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, err := s.getFromDisk(userID, mode, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance %s not found: %w", instanceID, err)
	}

	if v, ok := patch["symbol"].(string); ok && v != "" && v != inst.Symbol {
		return nil, fmt.Errorf("cannot change symbol via patch (got %q, have %q): delete and recreate the instance", v, inst.Symbol)
	}
	if v, ok := patch["strategy"].(string); ok && v != "" && v != inst.Strategy {
		return nil, fmt.Errorf("cannot change strategy via patch (got %q, have %q): delete and recreate the instance", v, inst.Strategy)
	}

	var base map[string]any
	if len(inst.Config) > 0 && string(inst.Config) != "null" {
		if err := json.Unmarshal(inst.Config, &base); err != nil {
			return nil, fmt.Errorf("parse existing config: %w", err)
		}
	}
	if base == nil {
		base = map[string]any{}
	}
	merged := deepMerge(base, patch)
	b, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("encode merged config: %w", err)
	}
	inst.Config = b

	if err := s.writeInstanceYAML(inst, hasCredentials); err != nil {
		return nil, err
	}
	return inst, nil
}

// alignConfigLeverage mirrors FuturesConfig.Leverage into the strategy params
// JSON so the stored config matches what the runtime actually applies. Without
// this, the DB row keeps the user's pre-sync value (e.g. leverage=1) while
// the session runs at FuturesConfig.Leverage (e.g. 3), making the stored
// config misleading and triggering leverage_mismatch warnings.
func alignConfigLeverage(inst *StrategyInstance) {
	if inst.FuturesConfig == nil || inst.FuturesConfig.Leverage <= 0 {
		return
	}
	var params map[string]any
	if len(inst.Config) == 0 || string(inst.Config) == "null" {
		params = map[string]any{}
	} else if err := json.Unmarshal(inst.Config, &params); err != nil {
		return
	}
	params["leverage"] = inst.FuturesConfig.Leverage
	if b, err := json.Marshal(params); err == nil {
		inst.Config = b
	}
}

// RemoveInstance removes an instance's directory, files, and Supabase row.
func (s *InstanceStore) RemoveInstance(userID, mode, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.InstanceDir(userID, mode, instanceID)
	err := os.RemoveAll(dir)

	s.deleteFromSupabase(userID, mode, instanceID)
	return err
}

// MarkInstanceError records the captured error message and timestamp on the
// strategy_instances row so the frontend can surface why a container is
// crashlooping instead of reporting a phantom-active container. No-op when
// Supabase is unavailable — the captured error is best-effort, not critical.
func (s *InstanceStore) MarkInstanceError(userID, mode, instanceID, errMsg string) {
	if s.sb == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	update := map[string]string{
		"last_error":    errMsg,
		"last_error_at": now,
		"updated_at":    now,
	}
	_, _, err := s.sb.client.From("strategy_instances").
		Update(update, "", "").
		Eq("user_id", userID).
		Eq("mode", mode).
		Eq("instance_id", instanceID).
		Execute()
	if err != nil {
		log.Printf("mark instance %s error to supabase: %v\n", instanceID, err)
	}
}

// ClearInstanceError nulls out last_error / last_error_at after a successful
// start so a previously-crashlooping instance is shown as healthy again.
func (s *InstanceStore) ClearInstanceError(userID, mode, instanceID string) {
	if s.sb == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	update := map[string]string{
		"last_error":    "",
		"last_error_at": "",
		"updated_at":    now,
	}
	_, _, err := s.sb.client.From("strategy_instances").
		Update(update, "", "").
		Eq("user_id", userID).
		Eq("mode", mode).
		Eq("instance_id", instanceID).
		Execute()
	if err != nil {
		log.Printf("clear instance %s error in supabase: %v\n", instanceID, err)
	}
}

// GetInstance returns a single instance. Prefers Supabase when available.
// When mode is empty, searches both live and paper modes.
func (s *InstanceStore) GetInstance(userID, mode, instanceID string) (*StrategyInstance, error) {
	if mode != "" {
		if s.sb != nil {
			return s.getFromSupabase(userID, mode, instanceID)
		}
		return s.getFromDisk(userID, mode, instanceID)
	}
	for _, m := range []string{ModeLive, ModePaper} {
		if s.sb != nil {
			inst, err := s.getFromSupabase(userID, m, instanceID)
			if err == nil {
				return inst, nil
			}
			continue
		}
		inst, err := s.getFromDisk(userID, m, instanceID)
		if err == nil {
			return inst, nil
		}
	}
	return nil, fmt.Errorf("instance %s not found", instanceID)
}

// ListInstances lists instances for a user/mode. Prefers Supabase when available.
func (s *InstanceStore) ListInstances(userID, mode string) ([]StrategyInstance, error) {
	if s.sb != nil {
		return s.listFromSupabase(userID, mode)
	}
	return s.listFromDisk(userID, mode)
}

// ListAllInstances discovers all instances across both modes for a user.
func (s *InstanceStore) ListAllInstances(userID string) ([]StrategyInstance, error) {
	var all []StrategyInstance
	for _, mode := range []string{ModeLive, ModePaper} {
		instances, err := s.ListInstances(userID, mode)
		if err != nil {
			return nil, err
		}
		all = append(all, instances...)
	}
	return all, nil
}

// YAMLExists returns true if a bbgo.yaml exists for the instance.
func (s *InstanceStore) YAMLExists(userID, mode, instanceID string) bool {
	_, err := os.Stat(s.yamlPath(userID, mode, instanceID))
	return err == nil
}

// ScanUsers scans DATA_DIR for user directories with instance configs.
func (s *InstanceStore) ScanUsers() []UserMode {
	var result []UserMode
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		userDir := filepath.Join(s.dataDir, e.Name())
		modeEntries, err := os.ReadDir(userDir)
		if err != nil {
			continue
		}
		for _, me := range modeEntries {
			if !me.IsDir() {
				continue
			}
			if me.Name() != ModeLive && me.Name() != ModePaper {
				continue
			}
			modeDir := filepath.Join(userDir, me.Name())
			instEntries, err := os.ReadDir(modeDir)
			if err != nil {
				continue
			}
			for _, ie := range instEntries {
				if !ie.IsDir() {
					continue
				}
				yamlPath := filepath.Join(modeDir, ie.Name(), "bbgo.yaml")
				if _, err := os.Stat(yamlPath); err == nil {
					result = append(result, UserMode{UserID: e.Name(), Mode: me.Name()})
					break
				}
			}
		}
	}
	return result
}

func (s *InstanceStore) listFromDisk(userID, mode string) ([]StrategyInstance, error) {
	modeDir := filepath.Join(s.dataDir, userID, mode)
	entries, err := os.ReadDir(modeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var instances []StrategyInstance
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		yamlPath := filepath.Join(modeDir, e.Name(), "bbgo.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			continue
		}
		inst, err := parseInstanceYAML(data, userID, mode, e.Name())
		if err != nil {
			continue
		}
		instances = append(instances, *inst)
	}
	return instances, nil
}

func (s *InstanceStore) getFromDisk(userID, mode, instanceID string) (*StrategyInstance, error) {
	data, err := os.ReadFile(s.yamlPath(userID, mode, instanceID))
	if err != nil {
		return nil, err
	}
	return parseInstanceYAML(data, userID, mode, instanceID)
}

// --- Supabase-backed methods ---

func (s *InstanceStore) upsertToSupabase(inst *StrategyInstance) {
	if s.supabaseUpsertFn != nil {
		s.supabaseUpsertFn(inst)
		return
	}
	if s.sb == nil {
		return
	}
	alignConfigLeverage(inst)
	config := json.RawMessage(`{}`)
	if len(inst.Config) > 0 && string(inst.Config) != "null" {
		config = inst.Config
	}
	var sessions any
	if inst.Sessions != nil {
		b, _ := json.Marshal(inst.Sessions)
		sessions = json.RawMessage(b)
	} else {
		sessions = json.RawMessage(`[]`)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	row := PublicStrategyInstancesInsert{
		InstanceId:    inst.InstanceID,
		UserId:        inst.UserID,
		Mode:          inst.Mode,
		Strategy:      inst.Strategy,
		Exchange:      &inst.Exchange,
		Symbol:        &inst.Symbol,
		Config:        config,
		Name:          &inst.Name,
		CrossExchange: &inst.CrossExchange,
		Sessions:      sessions,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if inst.FuturesConfig != nil {
		b, _ := json.Marshal(inst.FuturesConfig)
		row.FuturesConfig = json.RawMessage(b)
	}
	if inst.RiskConfig != nil && inst.RiskConfig.HasAny() {
		b, _ := json.Marshal(inst.RiskConfig)
		row.RiskConfig = json.RawMessage(b)
	}
	_, _, err := s.sb.client.From("strategy_instances").Upsert(row, "user_id,mode,instance_id", "", "").Execute()
	if err != nil {
		log.Printf("upsert instance %s to supabase: %v\n", inst.InstanceID, err)
	}
}

func (s *InstanceStore) deleteFromSupabase(userID, mode, instanceID string) {
	if s.sb == nil {
		return
	}
	_, _, err := s.sb.client.From("strategy_instances").
		Delete("", "").
		Eq("user_id", userID).
		Eq("mode", mode).
		Eq("instance_id", instanceID).
		Execute()
	if err != nil {
		log.Printf("delete instance %s from supabase: %v\n", instanceID, err)
	}

	// Cascade-delete orphaned trade data so a future bot reusing the same
	// deterministic instance_id (e.g. delete + recreate with same strategy/
	// symbol/interval) does not inherit the deleted bot's positions via
	// RestoreFromDB. Paper mode cleans paper_* tables; live mode cleans the
	// shared bbgo tables.
	tables := []string{
		"orders", "trades", "positions", "profits",
		"futures_position_risks",
		"margin_loans", "margin_repays", "margin_interests", "margin_liquidations",
		"nav_history_details", "rewards", "deposits", "withdraws",
	}
	prefix := ""
	if mode == ModePaper {
		prefix = "paper_"
	}
	for _, base := range tables {
		t := prefix + base
		_, _, derr := s.sb.client.From(t).
			Delete("", "").
			Eq("user_id", userID).
			Eq("strategy_instance_id", instanceID).
			Execute()
		if derr != nil {
			log.Printf("delete instance %s cascade from %s: %v\n", instanceID, t, derr)
		}
	}
	// paper_balances is keyed by user_id + strategy_instance_id + currency;
	// clear rows for this instance so the next bot starts from seed balances.
	if mode == ModePaper {
		_, _, derr := s.sb.client.From("paper_balances").
			Delete("", "").
			Eq("user_id", userID).
			Eq("strategy_instance_id", instanceID).
			Execute()
		if derr != nil {
			log.Printf("delete instance %s cascade from paper_balances: %v\n", instanceID, derr)
		}
	}
}

func (s *InstanceStore) listFromSupabase(userID, mode string) ([]StrategyInstance, error) {
	data, _, err := s.sb.client.From("strategy_instances").
		Select("*", "", false).
		Eq("user_id", userID).
		Eq("mode", mode).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("list instances from supabase: %w", err)
	}
	var rows []PublicStrategyInstancesSelect
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("decode instances: %w", err)
	}
	instances := make([]StrategyInstance, 0, len(rows))
	for _, r := range rows {
		inst := rowToInstance(r)
		instances = append(instances, inst)
	}
	return instances, nil
}

func (s *InstanceStore) getFromSupabase(userID, mode, instanceID string) (*StrategyInstance, error) {
	data, _, err := s.sb.client.From("strategy_instances").
		Select("*", "", false).
		Eq("user_id", userID).
		Eq("mode", mode).
		Eq("instance_id", instanceID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("get instance from supabase: %w", err)
	}
	var rows []PublicStrategyInstancesSelect
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("decode instance: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}
	inst := rowToInstance(rows[0])
	return &inst, nil
}

func rowToInstance(r PublicStrategyInstancesSelect) StrategyInstance {
	config, _ := json.Marshal(r.Config)
	var sessions []SessionRoleConfig
	if r.Sessions != nil {
		b, _ := json.Marshal(r.Sessions)
		json.Unmarshal(b, &sessions)
	}
	var futuresConfig *FuturesConfig
	if r.FuturesConfig != nil {
		b, _ := json.Marshal(r.FuturesConfig)
		json.Unmarshal(b, &futuresConfig)
	}
	var riskConfig *RiskConfig
	if r.RiskConfig != nil {
		b, _ := json.Marshal(r.RiskConfig)
		json.Unmarshal(b, &riskConfig)
	}
	return StrategyInstance{
		InstanceID:    r.InstanceId,
		UserID:        r.UserId,
		Mode:          r.Mode,
		Strategy:      r.Strategy,
		Exchange:      r.Exchange,
		Symbol:        r.Symbol,
		Config:        config,
		Name:          r.Name,
		CrossExchange: r.CrossExchange,
		Sessions:      sessions,
		FuturesConfig: futuresConfig,
		RiskConfig:    riskConfig,
		LastError:     derefStr(r.LastError),
		LastErrorAt:   derefStr(r.LastErrorAt),
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// parseInstanceYAML extracts a StrategyInstance from a single-strategy bbgo.yaml.
func parseInstanceYAML(data []byte, userID, mode, dirName string) (*StrategyInstance, error) {
	var cfg bbgoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse bbgo.yaml: %w", err)
	}

	inst := &StrategyInstance{
		UserID: userID,
		Mode:   mode,
	}

	for _, es := range cfg.ExchangeStrategies {
		entry, ok := parseExchangeStrategyEntry(es)
		if !ok {
			continue
		}
		inst.Strategy = entry.Strategy
		inst.Exchange = entry.Exchange
		inst.Config = entry.Config
		inst.Symbol = extractSymbolFromConfig(entry.Config)
		break
	}

	for _, cs := range cfg.CrossExchangeStrategies {
		entry, ok := parseCrossStrategyEntry(cs, cfg.Sessions)
		if !ok {
			continue
		}
		inst.Strategy = entry.Strategy
		inst.Config = entry.Config
		inst.CrossExchange = true
		inst.Sessions = entry.Sessions
		inst.Symbol = extractSymbolFromConfig(entry.Config)
		if len(entry.Sessions) > 0 {
			inst.Exchange = entry.Sessions[0].Exchange
		}
		break
	}

	if cfg.InstanceID != "" {
		inst.InstanceID = cfg.InstanceID
	} else if inst.Strategy != "" && inst.Symbol != "" {
		inst.InstanceID = computeInstanceID(inst.Strategy, inst.Symbol, inst.Config)
	} else {
		inst.InstanceID = dirName
	}

	// Extract futures config from session
	for _, sc := range cfg.Sessions {
		if sc.Futures {
			fc := &FuturesConfig{}
			if len(sc.SymbolLeverage) > 0 {
				for _, lev := range sc.SymbolLeverage {
					fc.Leverage = lev
					break
				}
			}
			if sc.IsolatedFutures {
				fc.MarginType = "isolated"
			} else {
				fc.MarginType = "cross"
			}
			inst.FuturesConfig = fc
			break
		}
	}

	return inst, nil
}

// buildInstanceYAML generates a bbgo YAML config for a single strategy instance.
func buildInstanceYAML(inst *StrategyInstance, hasCredentials func(string) bool, registry *StrategyDefaultsCache) ([]byte, error) {
	var params map[string]any
	if len(inst.Config) == 0 || string(inst.Config) == "null" {
		params = map[string]any{}
	} else if err := json.Unmarshal(inst.Config, &params); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if registry != nil {
		if defaults := registry.GetDefaults(inst.Strategy); defaults != nil {
			params = deepMerge(defaults, params)
		}
	}

	inst.Strategy, params = normalizeStrategyConfig(inst.Strategy, params)

	symbol := inst.Symbol
	if symbol == "" {
		if v, ok := params["symbol"].(string); ok && v != "" {
			symbol = v
		}
	}
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	params["symbol"] = symbol

	exchanges := map[string]exchangeConfig{}
	sessions := map[string]sessionConfig{}
	var exchangeStrategies []map[string]any
	var crossStrategies []map[string]any

	if inst.CrossExchange {
		for _, sr := range inst.Sessions {
			prefix := sr.EnvVarPrefix
			if prefix == "" {
				prefix = exchangeEnvPrefix(sr.Exchange)
			}
			sc := sessionConfig{
				Exchange:     sr.Exchange,
				EnvVarPrefix: prefix,
				Futures:      sr.Futures,
				PublicOnly:   !hasCredentials(sr.Exchange),
			}
			if inst.Mode == ModePaper {
				sc.PaperBalances = defaultPaperBalances
			}
			// Apply FuturesConfig (leverage + marginType) to each futures session role.
			// Previously this branch only set Futures=true and dropped leverage,
			// silently making cross-exchange futures strategies (xmaker, xfunding)
			// run with exchange defaults instead of user-chosen leverage.
			if sr.Futures && inst.FuturesConfig != nil {
				if inst.FuturesConfig.Leverage > 0 {
					sc.SymbolLeverage = map[string]int{symbol: inst.FuturesConfig.Leverage}
				}
				if inst.FuturesConfig.MarginType == "isolated" {
					sc.IsolatedFutures = true
					sc.IsolatedFuturesSymbol = symbol
				}
			}
			sessions[sr.Name] = sc
			if _, exists := exchanges[sr.Exchange]; !exists {
				exchanges[sr.Exchange] = exchangeConfig{Symbol: symbol}
			}
		}
		// Mirror FuturesConfig into strategy params so the strategy struct field
		// (e.g. xmaker.Leverage) is consistent with session symbolLeverage.
		if inst.FuturesConfig != nil {
			if inst.FuturesConfig.Leverage > 0 {
				params["leverage"] = inst.FuturesConfig.Leverage
			}
			if inst.FuturesConfig.MarginType != "" {
				params["marginType"] = inst.FuturesConfig.MarginType
			}
			// Single per-instance summary log reflecting what was actually applied
			// to futures sessions (fires once regardless of how many roles exist).
			log.Printf("instance %s (%s/%s): cross-exchange futures applied leverage=%d marginType=%s",
				inst.InstanceID, inst.Strategy, symbol, inst.FuturesConfig.Leverage, inst.FuturesConfig.MarginType)
		}
		crossStrategies = append(crossStrategies, map[string]any{
			inst.Strategy: params,
		})
	} else {
		exchange := inst.Exchange
		if exchange == "" {
			exchange = "binance"
		}
		exchanges[exchange] = exchangeConfig{Symbol: symbol}
		prefix := exchangeEnvPrefix(exchange)
		sc := sessionConfig{
			Exchange:     exchange,
			EnvVarPrefix: prefix,
			PublicOnly:   inst.Mode == ModePaper || !hasCredentials(exchange),
		}
		if inst.Mode == ModePaper {
			sc.PaperBalances = defaultPaperBalances
		}
		if registry != nil && registry.RequiresFutures(inst.Strategy) {
			sc.Futures = true
			leverage := 0
			marginType := ""
			if inst.FuturesConfig != nil {
				if inst.FuturesConfig.Leverage > 0 {
					leverage = inst.FuturesConfig.Leverage
				}
				marginType = inst.FuturesConfig.MarginType
			}
			if leverage == 0 {
				if lv := toFloat(params["leverage"]); lv >= 1 {
					leverage = int(lv)
				}
			} else {
				// Sync FuturesConfig.leverage into strategy params so the strategy
				// struct field (e.g. pivotshort.Leverage) sees the same value as
				// the session's symbolLeverage. Without this, session uses 10x
				// while the strategy struct falls back to its own default (3x),
				// causing mismatched position sizing and liquidation math.
				params["leverage"] = leverage
			}
			if marginType == "" {
				marginType, _ = params["marginType"].(string)
			} else {
				params["marginType"] = marginType
			}
			if leverage > 0 {
				sc.SymbolLeverage = map[string]int{symbol: leverage}
			}
			if marginType == "isolated" {
				sc.IsolatedFutures = true
				sc.IsolatedFuturesSymbol = symbol
			}
			log.Printf("instance %s (%s/%s): applying leverage=%d marginType=%s",
				inst.InstanceID, inst.Strategy, symbol, leverage, marginType)
		}
		sessions[exchange] = sc

		exchangeStrategies = append(exchangeStrategies, map[string]any{
			"on":          exchange,
			inst.Strategy: params,
		})
	}

	anyFutures := false
	for _, s := range sessions {
		if s.Futures {
			anyFutures = true
			break
		}
	}

	cfg := bbgoConfig{
		InstanceID: inst.InstanceID,
		Database:   nil,
		Sessions:   sessions,
		Exchange:   exchanges,
		Sync: &syncConfig{
			UserDataStream: &syncUserDataStreamConfig{
				Trades:                      true,
				FilledOrders:                true,
				FuturesPosition:             anyFutures,
				FuturesPositionSyncInterval: "30s",
			},
		},
		Persistence: &persistenceConfig{
			Json: &jsonPersistenceConfig{
				Directory: filepath.Join(ContainerDir(inst.UserID, inst.Mode, inst.InstanceID), "persistence"),
			},
		},
		ExchangeStrategies:      exchangeStrategies,
		CrossExchangeStrategies: crossStrategies,
	}
	cfg.Environment = &environmentConfig{}
	cfg.Environment.DisableStartupBalanceQuery = true
	if inst.Mode == ModePaper {
		cfg.Environment.PaperTrade = "1"
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal bbgo config: %w", err)
	}
	return out, nil
}

func extractSymbolFromConfig(config json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return ""
	}
	s, _ := m["symbol"].(string)
	return s
}


func parseExchangeStrategyEntry(m map[string]any) (StrategyEntry, bool) {
	exchange, _ := m["on"].(string)
	if exchange == "" {
		return StrategyEntry{}, false
	}
	for key, val := range m {
		if key == "on" {
			continue
		}
		config, err := json.Marshal(val)
		if err != nil {
			continue
		}
		return StrategyEntry{
			Exchange: exchange,
			Strategy: key,
			Config:   config,
		}, true
	}
	return StrategyEntry{}, false
}

func parseCrossStrategyEntry(m map[string]any, yamlSessions map[string]sessionConfig) (StrategyEntry, bool) {
	for key, val := range m {
		config, err := json.Marshal(val)
		if err != nil {
			continue
		}
		var sessions []SessionRoleConfig
		var names []string
		for name := range yamlSessions {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sc := yamlSessions[name]
			sessions = append(sessions, SessionRoleConfig{
				Name:         name,
				Exchange:     sc.Exchange,
				EnvVarPrefix: sc.EnvVarPrefix,
				Futures:      sc.Futures,
			})
		}
		return StrategyEntry{
			Strategy:      key,
			Config:        config,
			CrossExchange: true,
			Sessions:      sessions,
		}, true
	}
	return StrategyEntry{}, false
}
