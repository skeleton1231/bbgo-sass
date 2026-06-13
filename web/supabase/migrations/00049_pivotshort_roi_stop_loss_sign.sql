-- Pivotshort: roiStopLoss.percentage must be positive.
-- The exit executor (pkg/bbgo/exit_roi_stop_loss.go) treats the value as a
-- loss magnitude and negates it internally, so a stored -0.05 was being
-- double-negated and fired at breakeven. Migration 00029 originally wrote
-- -0.05; this corrects it to +0.05 to match documented semantics.
UPDATE strategy_registry
SET defaults = jsonb_set(
      defaults,
      '{exits,0,roiStopLoss,percentage}',
      '0.05'::jsonb
    ),
    updated_at = now()
WHERE id = 'pivotshort'
  AND defaults #>> '{exits,0,roiStopLoss,percentage}' IS DISTINCT FROM '0.05';
