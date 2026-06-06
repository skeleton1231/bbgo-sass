-- Strategy Registry Data Fixes
-- Makes 'defaults' column the complete bbgo config template for each strategy.
-- The manager deep-merges user form values on top of these defaults.
--
-- Also fixes live_only flags to match manager/user.go liveOnlyStrategies.

-- ============================================================
-- Fix 1: live_only flags (sync with manager/user.go liveOnlyStrategies)
-- ============================================================
UPDATE strategy_registry SET live_only = true WHERE id IN ('dca2', 'dca3', 'liquiditymaker', 'xhedgegrid');

-- ============================================================
-- Complete defaults templates (bbgo YAML config structure)
-- ============================================================

-- atrpin: ATR Pin strategy
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 14, "multiplier": 10, "quantity": 0.001
}'::jsonb WHERE id = 'atrpin';

-- audacitymaker: CRITICAL - orderFlow.quantity is required for trading
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 20,
  "orderFlow": {"interval": "1h", "quantity": 0.001}
}'::jsonb WHERE id = 'audacitymaker';

-- autoborrow: live-only margin management
UPDATE strategy_registry SET defaults = '{
  "interval": "5m", "minMarginLevel": 1.5, "maxMarginLevel": 3, "autoRepayWhenDeposit": true
}'::jsonb WHERE id = 'autoborrow';

-- autobuy: periodic buying with optional Bollinger filter
UPDATE strategy_registry SET defaults = '{
  "schedule": "0 10 * * *", "quantity": 0.001, "amount": 100,
  "minBaseBalance": 0, "dryRun": false,
  "bollinger": {"interval": "1h", "window": 20}
}'::jsonb WHERE id = 'autobuy';

-- bollgrid: Bollinger Band grid
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "gridNumber": 10, "gridPips": 50,
  "quantity": 0.001, "profitSpread": 50
}'::jsonb WHERE id = 'bollgrid';

-- bollmaker: Bollinger Band market maker with nested bollinger configs
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001,
  "spread": 0.001, "minProfitSpread": 0.001, "maxExposurePosition": 1,
  "disableShort": false, "tradeInBand": false, "shadowProtection": false,
  "defaultBollinger": {"interval": "1h", "window": 20},
  "neutralBollinger": {"interval": "1h", "window": 20}
}'::jsonb WHERE id = 'bollmaker';

-- convert: live-only asset converter
UPDATE strategy_registry SET defaults = '{
  "from": "BTC", "to": "USDT"
}'::jsonb WHERE id = 'convert';

-- dca: dollar cost averaging
UPDATE strategy_registry SET defaults = '{
  "investmentInterval": "1h", "budget": 500, "budgetPeriod": "day"
}'::jsonb WHERE id = 'dca';

-- dca2: advanced DCA (live-only)
UPDATE strategy_registry SET defaults = '{
  "quoteInvestment": 1000, "maxOrderCount": 5,
  "priceDeviation": 0.01, "takeProfitRatio": 0.05
}'::jsonb WHERE id = 'dca2';

-- dca3: DCA v3 variant (live-only)
UPDATE strategy_registry SET defaults = '{
  "quoteInvestment": 1000, "maxOrderCount": 5,
  "priceDeviation": 0.01, "takeProfitRatio": 0.05
}'::jsonb WHERE id = 'dca3';

-- deposit2transfer: live-only deposit watcher
UPDATE strategy_registry SET defaults = '{
  "assets": "USDT,BTC", "interval": "30s", "ignoreDust": true
}'::jsonb WHERE id = 'deposit2transfer';

-- drift: MA with linear regression
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 100, "quantity": 0.001,
  "stoploss": 0.02, "useStopLoss": true, "useAtr": false,
  "predictOffset": 10, "minInterval": "1h", "atrWindow": 14,
  "generateGraph": true
}'::jsonb WHERE id = 'drift';

-- elliottwave: EWO with ATR stops
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "stoploss": 0.02, "minInterval": "1h",
  "windowATR": 14, "windowQuick": 5, "windowSlow": 35,
  "useHeikinAshi": false, "drawGraph": true
}'::jsonb WHERE id = 'elliottwave';

-- emacross: EMA crossover with leverage-based sizing (quantity optional)
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "fastWindow": 7, "slowWindow": 25,
  "leverage": 1
}'::jsonb WHERE id = 'emacross';

-- ewo_dgtrd: EWO divergence trading
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "sigWin": 5, "stoploss": 0.02,
  "cciStochFilterHigh": 100, "ewoChangeFilterHigh": 0, "ewoChangeFilterLow": 0
}'::jsonb WHERE id = 'ewo_dgtrd';

-- factorzoo: multi-factor with nested linear config
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 20,
  "linear": {"interval": "1h"}
}'::jsonb WHERE id = 'factorzoo';

-- fixedmaker: simple fixed-spread maker
UPDATE strategy_registry SET defaults = '{
  "interval": "1m", "quantity": 0.001, "halfSpread": 0.001, "dryRun": false
}'::jsonb WHERE id = 'fixedmaker';

-- flashcrash: flash crash buyer
UPDATE strategy_registry SET defaults = '{
  "interval": "1m", "gridNumber": 10, "percentage": 0.01, "baseQuantity": 0.001
}'::jsonb WHERE id = 'flashcrash';

-- fmaker: flexible market maker
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "quantity": 0.001, "spread": 0.001, "minProfitSpread": 0.001
}'::jsonb WHERE id = 'fmaker';

-- grid: classic grid trading
UPDATE strategy_registry SET defaults = '{
  "gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000,
  "quantity": 0.001, "profitSpread": 50, "side": "both", "catchUp": false
}'::jsonb WHERE id = 'grid';

-- grid2: advanced grid with compound/stop-loss
UPDATE strategy_registry SET defaults = '{
  "gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000,
  "quantity": 0.001, "profitSpread": 0, "quoteInvestment": 1000,
  "compound": false, "earnBase": false,
  "triggerPrice": 0, "stopLossPrice": 0, "takeProfitPrice": 0
}'::jsonb WHERE id = 'grid2';

-- harmonic: SHARK pattern detection
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 20, "quantity": 0.001, "drawGraph": true
}'::jsonb WHERE id = 'harmonic';

-- irr: negative return rate mean-reversion
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 20, "quantity": 0.001, "drawGraph": true
}'::jsonb WHERE id = 'irr';

-- linregmaker: linear regression maker with nested indicator configs
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001,
  "spread": 0.001, "minProfitSpread": 0.001, "maxExposurePosition": 1,
  "reverseEMA": {"interval": "1h", "window": 100},
  "fastLinReg": {"interval": "1h", "window": 30},
  "slowLinReg": {"interval": "1h", "window": 60}
}'::jsonb WHERE id = 'linregmaker';

-- liquiditymaker: live-only advanced maker
UPDATE strategy_registry SET defaults = '{
  "numOfLiquidityLayers": 5, "spread": 0.001,
  "askLiquidityAmount": 0.001, "bidLiquidityAmount": 0.001,
  "liquidityPriceRange": 0.01, "maxPositionExposure": 1,
  "adjustmentUpdateInterval": "1h", "liquidityUpdateInterval": "1h"
}'::jsonb WHERE id = 'liquiditymaker';

-- pivotshort: pivot breakout short with nested breakLow config
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "quantity": 0.001, "leverage": 1,
  "breakLow": {"interval": "1h", "window": 7, "ratio": 0.01}
}'::jsonb WHERE id = 'pivotshort';

-- random: random trading for testing
UPDATE strategy_registry SET defaults = '{
  "schedule": "*/30 * * * *", "quantity": 0.001, "dryRun": true
}'::jsonb WHERE id = 'random';

-- rebalance: portfolio rebalancing
UPDATE strategy_registry SET defaults = '{
  "schedule": "0 */4 * * *", "quoteCurrency": "USDT",
  "threshold": 0.05, "dryRun": false
}'::jsonb WHERE id = 'rebalance';

-- rsmaker: RS-based maker with nested bollinger configs
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001,
  "spread": 0.001, "minProfitSpread": 0.001, "maxExposurePosition": 1,
  "neutralBollinger": {"interval": "1h", "window": 20, "bandWidth": 2.0},
  "defaultBollinger": {"interval": "1h", "window": 20, "bandWidth": 2.0}
}'::jsonb WHERE id = 'rsmaker';

-- schedule: scheduled order with optional MA conditions
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "side": "buy", "quantity": 0.001,
  "useLimitOrder": false, "minBaseBalance": 0, "maxBaseBalance": 0
}'::jsonb WHERE id = 'schedule';

-- scmaker: SC maker with nested indicator configs
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 20, "k": 0.5,
  "numOfLiquidityLayers": 5, "maxExposure": 1, "minProfit": 0.001,
  "adjustmentUpdateInterval": "1h", "liquidityUpdateInterval": "1h"
}'::jsonb WHERE id = 'scmaker';

-- sentinel: live-only anomaly monitor
UPDATE strategy_registry SET defaults = '{
  "interval": "1m", "threshold": 0.5, "window": 100,
  "numSamples": 50, "proportion": 0.1
}'::jsonb WHERE id = 'sentinel';

-- supertrend: trend following with window default from setupIndicators
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "window": 39, "supertrendMultiplier": 3,
  "quantity": 0.001, "leverage": 1,
  "takeProfitAtrMultiplier": 2, "stopByReversedSupertrend": false,
  "drawGraph": true
}'::jsonb WHERE id = 'supertrend';

-- support: support/resistance monitor
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "quantity": 0.001
}'::jsonb WHERE id = 'support';

-- swing: swing trading with MA filter
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "baseQuantity": 0.001, "minChange": 0.01,
  "movingAverageType": "SMA", "movingAverageInterval": "1h", "movingAverageWindow": 20
}'::jsonb WHERE id = 'swing';

-- techsignal: technical signal with nested supportDetection
UPDATE strategy_registry SET defaults = '{
  "interval": "1h",
  "supportDetection": [{"interval": "1h", "movingAverageInterval": "1h", "movingAverageWindow": 20, "movingAverageType": "SMA"}]
}'::jsonb WHERE id = 'techsignal';

-- trendtrader: trend line breakout with nested trendLine config
UPDATE strategy_registry SET defaults = '{
  "interval": "1h", "quantity": 0.001, "leverage": 1,
  "trendLine": {"interval": "1h", "quantity": 0.001, "pivotRightWindow": 5}
}'::jsonb WHERE id = 'trendtrader';

-- wall: large wall orders
UPDATE strategy_registry SET defaults = '{
  "side": "sell", "interval": "1m", "quantity": 0.1,
  "numLayers": 5, "layerSpread": 0.01
}'::jsonb WHERE id = 'wall';

-- xvs: volume surge with stop-loss
UPDATE strategy_registry SET defaults = '{
  "quantity": 0.001, "maxExposure": 1,
  "volumeInterval": "5m", "volumeThreshold": 800, "stoploss": 0.02
}'::jsonb WHERE id = 'xvs';

-- xhedgegrid: hedge grid (live-only, same as grid2 structure)
UPDATE strategy_registry SET defaults = '{
  "gridNumber": 10, "upperPrice": 70000, "lowerPrice": 50000,
  "quantity": 0.001, "profitSpread": 0, "quoteInvestment": 1000,
  "compound": false, "earnBase": false
}'::jsonb WHERE id = 'xhedgegrid';

-- cross-exchange strategies (minimal defaults, sessions handle most config)
UPDATE strategy_registry SET defaults = '{"interval": "1h"}'::jsonb WHERE id = 'xalign';
UPDATE strategy_registry SET defaults = '{"interval": "1h"}'::jsonb WHERE id = 'xbalance';
UPDATE strategy_registry SET defaults = '{"quantity": 0.001, "spread": 0.001}'::jsonb WHERE id = 'xdepthmaker';
UPDATE strategy_registry SET defaults = '{"halfSpread": 0.001, "quantity": 0.001}'::jsonb WHERE id = 'xfixedmaker';
UPDATE strategy_registry SET defaults = '{"quantity": 0.001}'::jsonb WHERE id = 'xfunding';
UPDATE strategy_registry SET defaults = '{"quantity": 0.001}'::jsonb WHERE id = 'xfundingv2';
UPDATE strategy_registry SET defaults = '{"quantity": 0.001}'::jsonb WHERE id = 'xgap';
UPDATE strategy_registry SET defaults = '{"quantity": 0.001, "spread": 0.001, "updateInterval": "1m", "hedgeInterval": "5m"}'::jsonb WHERE id = 'xmaker';
UPDATE strategy_registry SET defaults = '{"interval": "1h"}'::jsonb WHERE id = 'xnav';
UPDATE strategy_registry SET defaults = '{"interval": "5m", "quantity": 0.001}'::jsonb WHERE id = 'xpremium';

-- ============================================================
-- Fix emacross fields: make quantity optional, add leverage
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "fastWindow", "type": "number", "default": 9, "required": true, "label": "Fast EMA Period"},
  {"key": "slowWindow", "type": "number", "default": 21, "required": true, "label": "Slow EMA Period"},
  {"key": "quantity", "type": "number", "default": 0.001, "label": "Quantity", "description": "Optional: fixed quantity (omit for leverage-based sizing)"},
  {"key": "leverage", "type": "number", "default": 1, "label": "Leverage", "description": "Position leverage (used when quantity is not set)"}
]'::jsonb
WHERE id = 'emacross';

-- ============================================================
-- Fix supertrend fields: add window (default 39 from setupIndicators)
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "window", "type": "number", "default": 39, "label": "Window", "description": "Indicator window (supertrend period)"},
  {"key": "supertrendMultiplier", "type": "number", "default": 3, "label": "ATR Multiplier"},
  {"key": "quantity", "type": "number", "default": 0.001, "label": "Quantity"},
  {"key": "leverage", "type": "number", "default": 1, "label": "Leverage"},
  {"key": "takeProfitAtrMultiplier", "type": "number", "default": 2, "label": "Take Profit ATR Mult"},
  {"key": "stopByReversedSupertrend", "type": "boolean", "default": false, "label": "Stop on Reversal"},
  {"key": "drawGraph", "type": "boolean", "default": true, "label": "Draw Graph"}
]'::jsonb
WHERE id = 'supertrend';

-- ============================================================
-- Fix audacitymaker fields: add orderFlow quantity hint
-- ============================================================
UPDATE strategy_registry
SET fields = '[
  {"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"},
  {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"},
  {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window"},
  {"key": "orderFlowQuantity", "type": "number", "default": 0.001, "required": true, "label": "Order Flow Quantity", "description": "Quantity per order flow trade (stored in orderFlow.quantity)"}
]'::jsonb
WHERE id = 'audacitymaker';
