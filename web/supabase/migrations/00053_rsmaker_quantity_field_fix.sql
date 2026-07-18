-- Fix rsmaker: registry exposed bidQuantity/askQuantity, but the strategy struct
-- (pkg/strategy/rsmaker/strategy.go:42) embeds bbgo.QuantityOrAmount and has NO
-- bidQuantity/askQuantity fields. Validate() (strategy.go:157) rejects with
-- "quantity or amount must be > 0", so every rsmaker instance created via the SaaS
-- form/API crashed on startup. The bidQuantity/askQuantity were likely copied from
-- xmaker, which does have those fields.
--
-- Replace them with the real required field `quantity` (matching QuantityOrAmount).
-- Same UPDATE-fields-array pattern as 00043_bollmaker_bandwidth_fix.sql.

-- 1) Fix defaults: drop the bogus keys, add quantity.
UPDATE strategy_registry
SET defaults = (defaults - 'bidQuantity' - 'askQuantity') || '{"quantity": 0.001}'::jsonb
WHERE id = 'rsmaker';

-- 2) Fix fields: swap bidQuantity/askQuantity for a required quantity field.
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "required": true, "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "required": true, "label": "Quantity"},
  {"key": "spread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Spread"},
  {"key": "minProfitSpread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Min Profit Spread"},
  {"key": "neutralBollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Neutral Bollinger Interval", "group": "neutralBollinger"},
  {"key": "neutralBollinger.window", "type": "number", "default": 20, "min": 1, "label": "Neutral Bollinger Window", "group": "neutralBollinger"},
  {"key": "neutralBollinger.bandWidth", "type": "number", "default": 2.0, "min": 0.1, "label": "Neutral Bollinger Band Width", "group": "neutralBollinger"},
  {"key": "defaultBollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Default Bollinger Interval", "group": "defaultBollinger"},
  {"key": "defaultBollinger.window", "type": "number", "default": 20, "min": 1, "label": "Default Bollinger Window", "group": "defaultBollinger"},
  {"key": "defaultBollinger.bandWidth", "type": "number", "default": 2.0, "min": 0.1, "label": "Default Bollinger Band Width", "group": "defaultBollinger"}
]'::jsonb
WHERE id = 'rsmaker';
