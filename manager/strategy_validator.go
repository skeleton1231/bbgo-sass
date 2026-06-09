package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type StrategyWarning struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

// ValidateStrategyConfig checks a merged strategy config for problems.
// If registry fields are available, it uses data-driven required/type/min/max checks.
// Falls back to hardcoded per-strategy rules when no registry data exists.
func ValidateStrategyConfig(strategy string, config json.RawMessage) []StrategyWarning {
	var m map[string]any
	if len(config) == 0 || string(config) == "null" {
		m = map[string]any{}
	} else if err := json.Unmarshal(config, &m); err != nil {
		m = map[string]any{}
	}

	return validateWithRegistry(strategy, m)
}

// validateWithRegistry uses strategy_registry.fields for data-driven validation.
// Hardcoded per-strategy rules in validateFallback are used only as a safety net
// when no registry data is loaded (e.g., in unit tests with no Supabase).
func validateWithRegistry(strategy string, m map[string]any) []StrategyWarning {
	if registryFields := globalFieldsForTest[strategy]; len(registryFields) > 0 {
		return validateFromFields(strategy, m, registryFields)
	}
	if provider := globalFieldsProvider; provider != nil {
		if fields := provider.GetFields(strategy); len(fields) > 0 {
			return validateFromFields(strategy, m, fields)
		}
	}
	return validateFallback(strategy, m)
}

// validateFromFields is the data-driven validator: iterates FieldDef entries
// marked required=true and checks the merged config for presence and value.
func validateFromFields(strategy string, m map[string]any, fields []FieldDef) []StrategyWarning {
	var warnings []StrategyWarning

	for _, f := range fields {
		if !f.Required {
			continue
		}

		val, exists := lookupNested(m, f.Key)

		switch f.Type {
		case "number":
			if !exists || toFloat(val) <= 0 {
				warnings = append(warnings, StrategyWarning{
					ID:      fmt.Sprintf("missing_%s", toSnakeCase(f.Key)),
					Message: fmt.Sprintf("%s is required for %s", f.Key, strategy),
					Level:   "critical",
				})
			} else if f.Min != nil && toFloat(val) < *f.Min {
				warnings = append(warnings, StrategyWarning{
					ID:      fmt.Sprintf("invalid_%s", toSnakeCase(f.Key)),
					Message: fmt.Sprintf("%s must be >= %v", f.Key, *f.Min),
					Level:   "critical",
				})
			}
		case "text", "select":
			if !exists || toString(val) == "" {
				warnings = append(warnings, StrategyWarning{
					ID:      fmt.Sprintf("missing_%s", toSnakeCase(f.Key)),
					Message: fmt.Sprintf("%s is required for %s", f.Key, strategy),
					Level:   "critical",
				})
			}
		}
	}

	// Cross-cutting checks not expressible as field-level rules
	warnings = append(warnings, validateCrossCutting(strategy, m)...)

	return warnings
}

// validateCrossCutting handles multi-field invariants that can't be expressed
// as individual field required checks (e.g., upperPrice > lowerPrice).
func validateCrossCutting(strategy string, m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	switch strategy {
	case "grid", "grid2":
		upper := toFloat(m["upperPrice"])
		lower := toFloat(m["lowerPrice"])
		if upper > 0 && lower > 0 && upper <= lower {
			warnings = append(warnings, StrategyWarning{
				ID: "invalid_price_range", Message: "upperPrice must be greater than lowerPrice", Level: "critical",
			})
		}
	case "pivotshort":
		leverage := toFloat(m["leverage"])
		if leverage <= 0 {
			warnings = append(warnings, StrategyWarning{
				ID: "futures_no_leverage", Message: "pivotshort is a futures strategy — leverage must be set", Level: "critical",
			})
		}
	}

	return warnings
}

// globalFieldsForTest is set by tests to inject FieldDef data without Supabase.
var globalFieldsForTest map[string][]FieldDef

// FieldsProvider is implemented by StrategyDefaultsCache to supply field definitions.
type FieldsProvider interface {
	GetFields(strategyID string) []FieldDef
}

// globalFieldsProvider is set at startup from StrategyDefaultsCache.
var globalFieldsProvider FieldsProvider

// SetFieldsProvider wires the registry cache into the validator.
func SetFieldsProvider(p FieldsProvider) {
	globalFieldsProvider = p
}

// validateFallback contains hardcoded per-strategy rules used when no registry
// field definitions are available. These are a safety net only.
func validateFallback(strategy string, m map[string]any) []StrategyWarning {
	switch strategy {
	case "grid", "grid2":
		return validateGrid(m)
	case "bollgrid":
		return validateBollgrid(m)
	case "fixedmaker", "xfixedmaker":
		return validateFixedmaker(m)
	case "fmaker":
		return validateFmaker(m)
	case "bollmaker", "linregmaker", "rsmaker", "audacitymaker", "liquiditymaker":
		return validateBollmaker(m)
	case "scmaker":
		return validateScmaker(m)
	case "supertrend":
		return validateSupertrend(m)
	case "emacross":
		return validateEmacross(m)
	case "trendtrader":
		return validateTrendtrader(m)
	case "atrpin":
		return validateAtrpin(m)
	case "pivotshort":
		return validatePivotshort(m)
	case "swing":
		return validateSwing(m)
	case "ewo_dgtrd", "ewoDgtrd":
		return validateEwoDgtrd(m)
	case "harmonic":
		return validateHarmonic(m)
	case "dca":
		return validateDCA(m)
	case "schedule", "autobuy":
		return validateSchedule(m)
	case "random":
		return validateRandom(m)
	case "xmaker", "xcross":
		return validateCrossMaker(m)
	default:
		return nil
	}
}

// --- Hardcoded fallback validators ---

func validateGrid(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	upper := toFloat(m["upperPrice"])
	lower := toFloat(m["lowerPrice"])
	if upper > 0 && lower > 0 && upper <= lower {
		warnings = append(warnings, StrategyWarning{
			ID: "invalid_price_range", Message: "upperPrice must be greater than lowerPrice", Level: "critical",
		})
	}

	gridNum := toFloat(m["gridNumber"])
	if gridNum < 2 {
		warnings = append(warnings, StrategyWarning{
			ID: "grid_too_few", Message: "gridNumber must be at least 2", Level: "critical",
		})
	}

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity, baseQuantity, or quoteInvestment is required", Level: "critical",
		})
	}

	profitSpread := toFloat(m["profitSpread"])
	if profitSpread <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_profit_spread", Message: "profitSpread is required for grid strategies to capture profit per grid level", Level: "warning",
		})
	}

	return warnings
}

func validateBollgrid(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	gridPips := toFloat(m["gridPips"])
	if gridPips <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_grid_pips", Message: "gridPips is required for bollgrid", Level: "critical",
		})
	}

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateFixedmaker(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	spread := toFloat(m["halfSpread"])
	if spread <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_spread", Message: "halfSpread is required for fixedmaker", Level: "critical",
		})
	}

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateFmaker(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	spread := toFloat(m["spread"])
	if spread <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_spread", Message: "spread is required for fmaker", Level: "critical",
		})
	}

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateBollmaker(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasMakerQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "bidQuantity/askQuantity or quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateScmaker(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateSupertrend(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required for supertrend", Level: "critical",
		})
	}

	return warnings
}

func validateEmacross(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	fw := toFloat(m["fastWindow"])
	if fw <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_fast_window", Message: "fastWindow is required (e.g. 7 or 9)", Level: "critical",
		})
	}

	sw := toFloat(m["slowWindow"])
	if sw <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_slow_window", Message: "slowWindow is required (e.g. 21 or 25)", Level: "critical",
		})
	}

	return warnings
}

func validateTrendtrader(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateAtrpin(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	k := toFloat(m["k"])
	if k <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_k", Message: "k (ATR multiplier) is required", Level: "critical",
		})
	}

	return warnings
}

func validatePivotshort(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	interval := toString(m["interval"])
	if interval == "" {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_interval", Message: "interval is required for pivotshort", Level: "critical",
		})
	}

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	leverage := toFloat(m["leverage"])
	if leverage <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "futures_no_leverage", Message: "pivotshort is a futures strategy — leverage must be set", Level: "critical",
		})
	}

	return warnings
}

func validateSwing(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	maWindow := toFloat(m["maWindow"])
	if maWindow <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_ma_window", Message: "maWindow is required for swing strategy", Level: "critical",
		})
	}

	return warnings
}

func validateEwoDgtrd(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateHarmonic(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
		})
	}

	return warnings
}

func validateDCA(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	budget := toFloat(m["budget"])
	if budget <= 0 {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_budget", Message: "budget is required for DCA", Level: "critical",
		})
	}

	return warnings
}

func validateSchedule(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	side := toString(m["side"])
	if side == "" {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_side", Message: "side (buy or sell) is required for schedule", Level: "critical",
		})
	}

	return warnings
}

func validateRandom(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	sched := toString(m["schedule"])
	if sched == "" {
		interval := toString(m["interval"])
		if interval == "" {
			warnings = append(warnings, StrategyWarning{
				ID: "missing_schedule", Message: "schedule or interval is required for random strategy", Level: "critical",
			})
		}
	}

	return warnings
}

func validateCrossMaker(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required for cross-exchange maker", Level: "critical",
		})
	}

	return warnings
}

// --- Helpers ---

func hasAnyQuantity(m map[string]any) bool {
	for _, key := range []string{"quantity", "baseQuantity", "quoteInvestment", "amount", "investAmount"} {
		if toFloat(m[key]) > 0 {
			return true
		}
	}
	return false
}

func hasMakerQuantity(m map[string]any) bool {
	if toFloat(m["bidQuantity"]) > 0 || toFloat(m["askQuantity"]) > 0 {
		return true
	}
	return hasAnyQuantity(m)
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		if !math.IsNaN(n) && !math.IsInf(n, 0) {
			return n
		}
		return 0
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0
		}
		return f
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}

// lookupNested resolves dot-separated keys like "breakLow.interval" in a nested map.
func lookupNested(m map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var current any = m
	for _, p := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = cm[p]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// toSnakeCase converts camelCase to snake_case for warning IDs.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
