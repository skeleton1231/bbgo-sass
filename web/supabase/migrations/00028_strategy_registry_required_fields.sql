-- Strategy Registry: add missing required=true flags to fields
-- Based on bbgo strategy source audit: these fields are mandatory for the
-- strategy to actually place orders after startup.

-- grid2: needs quantity OR quoteInvestment (at least one)
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'quantity' THEN elem || '{"required": true}'
         WHEN elem->>'key' = 'quoteInvestment' THEN elem || '{"required": true}'
         ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'grid2';

-- pivotshort: interval is needed for kline subscription
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'pivotshort';

-- swing: movingAverageWindow is mandatory
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'movingAverageWindow' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'swing';

-- audacitymaker: needs order quantity to trade
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'quantity' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'audacitymaker';

-- elliottwave: needs quantity for order placement
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'quantity' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'elliottwave';

-- ewo_dgtrd: needs quantity for order placement
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'quantity' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'ewo_dgtrd';

-- factorzoo: needs quantity for order placement
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'quantity' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'factorzoo';

-- random: needs schedule to trigger trades
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'schedule' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'random';

-- supertrend: interval needed for kline
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'supertrend';

-- emacross: interval needed for kline
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'emacross';

-- bollmaker family: interval needed for indicator
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id IN ('bollmaker', 'linregmaker', 'rsmaker');

-- fixedmaker family: interval needed
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id IN ('fixedmaker', 'xfixedmaker', 'fmaker');

-- grid: interval needed
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'grid';

-- bollgrid: interval needed
UPDATE strategy_registry SET fields = (
  SELECT jsonb_agg(
    CASE WHEN elem->>'key' = 'interval' THEN elem || '{"required": true}' ELSE elem END
  ) FROM jsonb_array_elements(fields) elem
) WHERE id = 'bollgrid';
