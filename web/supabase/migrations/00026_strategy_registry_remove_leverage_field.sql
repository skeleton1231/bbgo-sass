-- Strategy Registry: remove 'leverage' field from strategies that have requires_futures = true
-- The FuturesConfigFields component in the UI already handles leverage + marginType,
-- so having 'leverage' as a strategy field creates a duplicate entry.

UPDATE strategy_registry
SET fields = (
  SELECT jsonb_agg(elem)
  FROM jsonb_array_elements(fields) elem
  WHERE elem->>'key' != 'leverage'
)
WHERE requires_futures = true;
