-- Phase 3: unblock 6 futures-class strategies from paper mode.
--
-- These were flagged live_only before paper-futures existed. They now pass
-- the paper smoke gate (pkg/cmd/strategy.TestPaperSmoke_AllStrategies_RunWithoutPanic)
-- and their non-liveOnly futures siblings (pivotshort, fixedmaker) already
-- run in paper-futures. Verified paper-ready 2026-07-19.
--
-- Keep in sync with:
--   - saas/manager/strategy_types.go StrategyRegistry (LiveOnly: false for these)
--   - saas/manager/api_cross_layer_type_test.go testRegistry.liveOnly (removed)
--   - pkg/cmd/strategy/paper_smoke_test.go (these are NOT in paperSmokeSkip)

UPDATE strategy_registry SET live_only = false
WHERE id IN (
  'audacitymaker', 'bollmaker', 'drift', 'elliottwave', 'linregmaker', 'rsmaker'
);
