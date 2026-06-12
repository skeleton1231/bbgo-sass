-- Align strategy_registry.defaults.leverage with bbgo Defaults() for futures strategies.
-- Source of truth: pkg/strategy/<name>/strategy.go Defaults() method.
--
-- The deep-merge in manager/user.go:deepMerge layers registry defaults under user config,
-- so this value is only used when the user hasn't set leverage via FuturesConfig.
-- Once FuturesConfig.leverage is injected into config (api.go CreateStrategy handler),
-- it overrides this default.
--
-- bbgo Defaults() values:
--   pivotshort: 3   (pkg/strategy/pivotshort/strategy.go:124)
--   drift:      1   (pkg/strategy/drift/strategy.go:760)
--   xfunding:   1   (pkg/strategy/xfunding/strategy.go:197)
--   all others: 1   (no explicit default; 1 is the safe minimum)

-- pivotshort: bbgo defaults to 3x
UPDATE strategy_registry
SET defaults = jsonb_set(defaults, '{leverage}', '3'::jsonb, true)
WHERE id = 'pivotshort';

-- All other requires_futures strategies: default to 1x
UPDATE strategy_registry
SET defaults = jsonb_set(defaults, '{leverage}', '1'::jsonb, true)
WHERE requires_futures = true
  AND id != 'pivotshort';
