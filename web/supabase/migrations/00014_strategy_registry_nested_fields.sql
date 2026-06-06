-- Strategy Registry: nested sub-fields for strategies with dot-notation keys
-- Fields with dot-notation keys (e.g. "breakLow.interval") are auto-nested
-- into objects by the frontend nestConfig() before sending to the backend.

-- ============================================================
-- pivotshort: expose breakLow sub-fields
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Quantity"},
  {"key": "leverage", "type": "number", "default": 1, "min": 1, "max": 125, "label": "Leverage"},
  {"key": "breakLow.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Break Low Interval", "group": "breakLow"},
  {"key": "breakLow.window", "type": "number", "default": 7, "min": 1, "label": "Break Low Window", "group": "breakLow"},
  {"key": "breakLow.ratio", "type": "number", "default": 0.01, "min": 0.001, "max": 1, "label": "Break Low Ratio", "group": "breakLow"}
]'::jsonb
WHERE id = 'pivotshort';

-- ============================================================
-- trendtrader: expose trendLine sub-fields
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Quantity"},
  {"key": "leverage", "type": "number", "default": 1, "min": 1, "max": 125, "label": "Leverage"},
  {"key": "trendLine.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Trend Line Interval", "group": "trendLine"},
  {"key": "trendLine.quantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Trend Line Quantity", "group": "trendLine"},
  {"key": "trendLine.pivotRightWindow", "type": "number", "default": 5, "min": 1, "label": "Pivot Right Window", "group": "trendLine"}
]'::jsonb
WHERE id = 'trendtrader';

-- ============================================================
-- audacitymaker: expose orderFlow sub-fields (replace flat orderFlowQuantity)
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window"},
  {"key": "orderFlow.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Order Flow Interval", "group": "orderFlow"},
  {"key": "orderFlow.quantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Order Flow Quantity", "group": "orderFlow"}
]'::jsonb
WHERE id = 'audacitymaker';

-- ============================================================
-- bollmaker: expose bollinger sub-fields
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "bidQuantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Bid Quantity"},
  {"key": "askQuantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Ask Quantity"},
  {"key": "spread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Spread"},
  {"key": "minProfitSpread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Min Profit Spread"},
  {"key": "defaultBollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Default Bollinger Interval", "group": "defaultBollinger"},
  {"key": "defaultBollinger.window", "type": "number", "default": 20, "min": 1, "label": "Default Bollinger Window", "group": "defaultBollinger"},
  {"key": "neutralBollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Neutral Bollinger Interval", "group": "neutralBollinger"},
  {"key": "neutralBollinger.window", "type": "number", "default": 20, "min": 1, "label": "Neutral Bollinger Window", "group": "neutralBollinger"}
]'::jsonb
WHERE id = 'bollmaker';

-- ============================================================
-- linregmaker: expose indicator sub-fields
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "bidQuantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Bid Quantity"},
  {"key": "askQuantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Ask Quantity"},
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

-- ============================================================
-- rsmaker: expose bollinger sub-fields
-- ============================================================
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
  {"key": "defaultBollinger.bandWidth", "type": "number", "default": 2.0, "min": 0.1, "label": "Default Bollinger Band Width", "group": "defaultBollinger"}
]'::jsonb
WHERE id = 'rsmaker';

-- ============================================================
-- autobuy: expose bollinger sub-fields
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "schedule", "type": "text", "default": "0 10 * * *", "required": true, "label": "Schedule (Cron)"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Quantity"},
  {"key": "amount", "type": "number", "default": 100, "min": 1, "label": "Amount (Quote)"},
  {"key": "bollinger.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Bollinger Interval", "group": "bollinger"},
  {"key": "bollinger.window", "type": "number", "default": 20, "min": 1, "label": "Bollinger Window", "group": "bollinger"}
]'::jsonb
WHERE id = 'autobuy';

-- ============================================================
-- factorzoo: expose linear sub-fields
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "window", "type": "number", "default": 20, "min": 1, "label": "Window"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Quantity"},
  {"key": "linear.interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Linear Interval", "group": "linear"}
]'::jsonb
WHERE id = 'factorzoo';
