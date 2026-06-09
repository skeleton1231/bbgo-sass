package main

import (
	"encoding/json"
	"math"
)

type StrategyWarning struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

func ValidateStrategyConfig(strategy string, config json.RawMessage) []StrategyWarning {
	var m map[string]any
	if len(config) == 0 || string(config) == "null" {
		m = map[string]any{}
	} else if err := json.Unmarshal(config, &m); err != nil {
		m = map[string]any{}
	}

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
	case "drift":
		return validateDrift(m)
	case "elliottwave":
		return validateElliottwave(m)
	case "pivotshort":
		return validatePivotshort(m)
	case "swing":
		return validateSwing(m)
	case "ewo_dgtrd", "ewoDgtrd":
		return validateEwoDgtrd(m)
	case "harmonic":
		return validateHarmonic(m)
	case "irr":
		return validateIrr(m)
	case "dca":
		return validateDCA(m)
	case "schedule", "autobuy":
		return validateSchedule(m)
	case "random":
		return validateRandom(m)
	case "flashcrash":
		return validateFlashcrash(m)
	case "xmaker", "xcross":
		return validateCrossMaker(m)
	case "xlog":
		return validateXlog(m)
	case "xbalance":
		return validateXbalance(m)
	default:
		return nil
	}
}

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

func validateDrift(m map[string]any) []StrategyWarning {
	return nil
}

func validateElliottwave(m map[string]any) []StrategyWarning {
	var warnings []StrategyWarning

	if !hasAnyQuantity(m) {
		warnings = append(warnings, StrategyWarning{
			ID: "missing_quantity", Message: "quantity is required", Level: "critical",
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

func validateIrr(m map[string]any) []StrategyWarning {
	return nil
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

func validateFlashcrash(m map[string]any) []StrategyWarning {
	return nil
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

func validateXlog(m map[string]any) []StrategyWarning {
	return nil
}

func validateXbalance(m map[string]any) []StrategyWarning {
	return nil
}

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
