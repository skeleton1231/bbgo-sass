-- Fix linregmaker: same bug class as rsmaker (00053). The strategy struct
-- (pkg/strategy/linregmaker/strategy.go:115) embeds bbgo.QuantityOrAmount and has
-- NO bidQuantity/askQuantity fields, but the registry exposed bidQuantity/askQuantity.
-- Validate() doesn't check quantity (so it doesn't crash like rsmaker did), but the
-- strategy silently idles — QuantityOrAmount.CalculateQuantity returns 0, so it never
-- places real orders. Replace bidQuantity/askQuantity with the real `quantity` field.

-- 1) Fix defaults: drop bogus keys, add quantity.
UPDATE strategy_registry
SET defaults = (defaults - 'bidQuantity' - 'askQuantity') || '{"quantity": 0.001}'::jsonb
WHERE id = 'linregmaker';

-- 2) Fix fields: swap bidQuantity/askQuantity for a required quantity field.
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "required": true, "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "required": true, "label": "Quantity"},
  {"key": "spread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Spread"},
  {"key": "minProfitSpread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Min Profit Spread"},
  {"key": "reverseEMA.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Reverse EMA Interval", "group": "reverseEMA"},
  {"key": "reverseEMA.window", "type": "number", "default": 100, "min": 1, "label": "Reverse EMA Window", "group": "reverseEMA"},
  {"key": "fastLinReg.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Fast LinReg Interval", "group": "fastLinReg"},
  {"key": "fastLinReg.window", "type": "number", "default": 30, "min": 1, "label": "Fast LinReg Window", "group": "fastLinReg"},
  {"key": "slowLinReg.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Slow LinReg Interval", "group": "slowLinReg"},
  {"key": "slowLinReg.window", "type": "number", "default": 60, "min": 1, "label": "Slow LinReg Window", "group": "slowLinReg"}
]'::jsonb
WHERE id = 'linregmaker';
