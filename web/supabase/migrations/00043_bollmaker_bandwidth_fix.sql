-- Fix bollmaker: bollinger defaults missing bandWidth, which collapses BOLL bands to SMA.
-- Same root cause as 00016_autobuy_bollinger_bandwidth_fix.sql, but for bollmaker's
-- nested defaultBollinger/neutralBollinger objects. rsmaker (migration 00012 line 191-192)
-- already has bandWidth set correctly; bollmaker was missed.
--
-- bbgo Defaults() expects 3.0 for DefaultBollinger and 2.0 for NeutralBollinger,
-- matching the original in pkg/strategy/bollmaker/strategy.go.
UPDATE strategy_registry SET defaults = jsonb_set(
  jsonb_set(
    defaults,
    '{defaultBollinger}',
    '{"interval": "1h", "window": 20, "bandWidth": 3.0}'::jsonb,
    true
  ),
  '{neutralBollinger}',
  '{"interval": "1h", "window": 20, "bandWidth": 2.0}'::jsonb,
  true
)
WHERE id = 'bollmaker';

-- Expose bandWidth sub-fields in the form so users can tune them.
-- Mirrors the rsmaker field layout from migration 00014.
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "bidQuantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Bid Quantity"},
  {"key": "askQuantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Ask Quantity"},
  {"key": "spread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Spread"},
  {"key": "minProfitSpread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Min Profit Spread"},
  {"key": "neutralBollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Neutral Bollinger Interval", "group": "neutralBollinger"},
  {"key": "neutralBollinger.window", "type": "number", "default": 20, "min": 1, "label": "Neutral Bollinger Window", "group": "neutralBollinger"},
  {"key": "neutralBollinger.bandWidth", "type": "number", "default": 2.0, "min": 0.1, "label": "Neutral Bollinger Band Width", "group": "neutralBollinger"},
  {"key": "defaultBollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Default Bollinger Interval", "group": "defaultBollinger"},
  {"key": "defaultBollinger.window", "type": "number", "default": 20, "min": 1, "label": "Default Bollinger Window", "group": "defaultBollinger"},
  {"key": "defaultBollinger.bandWidth", "type": "number", "default": 3.0, "min": 0.1, "label": "Default Bollinger Band Width", "group": "defaultBollinger"}
]'::jsonb
WHERE id = 'bollmaker';
