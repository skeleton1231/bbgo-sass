-- Strategy Registry Fields Audit
-- Fixes from source code audit comparing bbgo strategy structs to registry data.

-- ============================================================
-- Fix 1: drift MinInterval casing
-- bbgo source json tag is "MinInterval" (capital M), not "minInterval"
-- Without this fix bbgo ignores the config value entirely
-- ============================================================
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 100, "quantity": 0.001,
  "stoploss": 0.02, "useStopLoss": true, "useAtr": false,
  "predictOffset": 10, "MinInterval": "1h", "atrWindow": 14,
  "generateGraph": true
}'::jsonb WHERE id = 'drift';

-- ============================================================
-- Fix 2: emacross defaults sync with fields
-- fields say fastWindow=9, slowWindow=21 but defaults had 7/25
-- ============================================================
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "fastWindow": 9, "slowWindow": 21,
  "leverage": 1
}'::jsonb WHERE id = 'emacross';

-- ============================================================
-- Fix 3: dca investmentInterval
-- strategy Defaults() sets "1d", not "1h"
-- ============================================================
UPDATE strategy_registry SET defaults = '{
  "investmentInterval": "1d", "budget": 500, "budgetPeriod": "day"
}'::jsonb WHERE id = 'dca';

-- ============================================================
-- Fix 4: techsignal remove stray top-level interval
-- Source has no top-level interval field (only inside supportDetection[])
-- ============================================================
UPDATE strategy_registry SET defaults = '{
  "supportDetection": [{"interval": "1h", "movingAverageInterval": "1h", "movingAverageWindow": 20, "movingAverageType": "SMA"}]
}'::jsonb WHERE id = 'techsignal';

-- ============================================================
-- Fix 5: bollgrid min/max constraints
-- Validate() requires profitSpread > 0 and quantity > 0
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "gridNumber", "type": "number", "default": 10, "min": 1, "label": "Grid Number"},
  {"key": "gridPips", "type": "number", "default": 50, "min": 1, "label": "Grid Pips"},
  {"key": "quantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Quantity"},
  {"key": "profitSpread", "type": "number", "default": 50, "required": true, "min": 0.01, "label": "Profit Spread"}
]'::jsonb
WHERE id = 'bollgrid';

-- ============================================================
-- Fix 6: grid min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "gridNumber", "type": "number", "default": 10, "min": 1, "label": "Grid Number"},
  {"key": "upperPrice", "type": "number", "default": 70000, "required": true, "min": 0, "label": "Upper Price"},
  {"key": "lowerPrice", "type": "number", "default": 50000, "required": true, "min": 0, "label": "Lower Price"},
  {"key": "quantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Quantity"},
  {"key": "profitSpread", "type": "number", "default": 50, "min": 0, "label": "Profit Spread"},
  {"key": "side", "type": "select", "default": "both", "options": ["buy", "sell", "both"], "label": "Side"}
]'::jsonb
WHERE id = 'grid';

-- ============================================================
-- Fix 7: grid2 min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "gridNumber", "type": "number", "default": 10, "min": 1, "label": "Grid Number"},
  {"key": "upperPrice", "type": "number", "default": 70000, "required": true, "min": 0, "label": "Upper Price"},
  {"key": "lowerPrice", "type": "number", "default": 50000, "required": true, "min": 0, "label": "Lower Price"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Quantity"},
  {"key": "profitSpread", "type": "number", "default": 0, "min": 0, "label": "Profit Spread"},
  {"key": "quoteInvestment", "type": "number", "default": 1000, "min": 0, "label": "Quote Investment"},
  {"key": "compound", "type": "boolean", "default": false, "label": "Compound"},
  {"key": "stopLossPrice", "type": "number", "default": 0, "min": 0, "label": "Stop Loss Price"}
]'::jsonb
WHERE id = 'grid2';

-- ============================================================
-- Fix 8: flashcrash min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1m", "options": ["1m", "5m", "15m"], "label": "Interval"},
  {"key": "gridNumber", "type": "number", "default": 10, "min": 1, "label": "Grid Number"},
  {"key": "percentage", "type": "number", "default": 0.01, "min": 0.001, "max": 0.5, "label": "Drop Percentage"},
  {"key": "baseQuantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Base Quantity"}
]'::jsonb
WHERE id = 'flashcrash';

-- ============================================================
-- Fix 9: fixedmaker min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1m", "options": ["1m", "5m", "15m"], "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Quantity"},
  {"key": "halfSpread", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Half Spread"}
]'::jsonb
WHERE id = 'fixedmaker';

-- ============================================================
-- Fix 10: fmaker min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.001, "min": 0.000001, "label": "Quantity"},
  {"key": "spread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Spread"},
  {"key": "minProfitSpread", "type": "number", "default": 0.001, "min": 0.000001, "label": "Min Profit Spread"}
]'::jsonb
WHERE id = 'fmaker';

-- ============================================================
-- Fix 11: wall min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "side", "type": "select", "default": "sell", "options": ["buy", "sell"], "label": "Side"},
  {"key": "interval", "type": "select", "default": "1m", "options": ["1m", "5m", "15m"], "label": "Interval"},
  {"key": "quantity", "type": "number", "default": 0.1, "required": true, "min": 0.000001, "label": "Quantity"},
  {"key": "numLayers", "type": "number", "default": 5, "min": 1, "max": 20, "label": "Number of Layers"},
  {"key": "layerSpread", "type": "number", "default": 0.01, "min": 0.000001, "label": "Layer Spread"}
]'::jsonb
WHERE id = 'wall';

-- ============================================================
-- Fix 12: xvs min/max constraints
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "quantity", "type": "number", "default": 0.001, "required": true, "min": 0.000001, "label": "Quantity"},
  {"key": "maxExposure", "type": "number", "default": 1, "min": 0.1, "label": "Max Exposure"},
  {"key": "volumeInterval", "type": "select", "default": "5m", "options": ["1m", "5m", "15m"], "label": "Volume Interval"},
  {"key": "volumeThreshold", "type": "number", "default": 800, "min": 1, "label": "Volume Threshold"},
  {"key": "stoploss", "type": "number", "default": 0.02, "min": 0.001, "max": 0.5, "label": "Stop Loss"}
]'::jsonb
WHERE id = 'xvs';
