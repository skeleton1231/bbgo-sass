-- Paper-mode live_only reconciliation + cross-exchange safety net.
--
-- Background (see saas/docs/paper-strategy-audit.md, finding #2 + #3):
--
-- 1. Migration ↔ DB drift. Migrations 00010 + 00012 seed only 8 live_only=true
--    rows, but the runtime authoritative set is ~23 (the manager's
--    StrategyDefaultsCache reads whatever live_only the DB holds). The
--    remainder was edited out-of-band, so a FRESH database (migrations only)
--    does NOT reproduce production's live_only set. This migration makes the
--    seed match the authoritative set mirrored by
--    saas/manager/api_cross_layer_type_test.go testRegistry (which the
--    TestCrossLayerLiveOnlyAlignment test asserts mirrors the hardcoded
--    StrategyRegistry const).
--
-- 2. Cross-exchange gray zone. 8 cross-exchange strategies were NOT live_only
--    and the manager allowed them into paper mode (binance-only), but
--    PaperTradeExchange wraps a SINGLE exchange and cannot truly support
--    multi-session cross-exchange strategies — so they silently malfunction
--    in paper. Until Phase 4 (multi-session paper) lands, block them from
--    paper via live_only. They remain fully usable in live mode.
--
-- Keep this list in sync with:
--   - saas/manager StrategyRegistry const (LiveOnly fields)
--   - saas/manager/api_cross_layer_type_test.go testRegistry.liveOnly
--   - pkg/cmd/strategy/paper_smoke_test.go paperSmokeSkip (cross-exchange reasons)
--
-- Phase 3 will flip specific rows back to false as the paper engine gains
-- support (futures-class, account-feed, etc.) — each via its own migration.

-- ============================================================
-- A. Cross-exchange safety net (decision 3): block from paper until Phase 4.
--    Not previously live_only. Live mode unaffected.
-- ============================================================
UPDATE strategy_registry SET live_only = true
WHERE id IN (
  'xalign', 'xbalance', 'xdepthmaker', 'xfixedmaker',
  'xfunding', 'xfundingv2', 'xgap', 'xmaker'
);

-- ============================================================
-- B. live_only reconciliation (decision 1): make the seed reproduce the
--    authoritative production set. These 23 mirror testRegistry. Idempotent
--    on the current (drifted) DB; the value is that a fresh DB now matches.
-- ============================================================
UPDATE strategy_registry SET live_only = true
WHERE id IN (
  'audacitymaker', 'autoborrow', 'autobuy', 'bollmaker', 'convert',
  'dca2', 'dca3', 'deposit2transfer', 'drift', 'elliottwave',
  'factorzoo', 'linregmaker', 'liquiditymaker', 'rebalance', 'rsmaker',
  'scmaker', 'sentinel', 'supertrend', 'support', 'wall',
  'xnav', 'xpremium', 'xvs'
);
