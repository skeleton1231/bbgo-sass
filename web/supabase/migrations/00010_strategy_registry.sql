-- Strategy Registry: single source of truth for all bbgo SaaS strategies
-- Generated from bbgo source audit + frontend definitions

CREATE TABLE IF NOT EXISTS strategy_registry (
  id             text PRIMARY KEY,
  display_name   text NOT NULL,
  description    text DEFAULT '',
  category       text NOT NULL DEFAULT 'other',
  exchanges      jsonb DEFAULT '[]',
  live_only      boolean DEFAULT false,
  cross_exchange boolean DEFAULT false,
  defaults       jsonb DEFAULT '{}',
  fields         jsonb DEFAULT '[]',
  session_roles  jsonb DEFAULT '[]',
  sort_order     int DEFAULT 0,
  enabled        boolean DEFAULT true,
  created_at     timestamptz DEFAULT now(),
  updated_at     timestamptz DEFAULT now()
);

-- Enable RLS
ALTER TABLE strategy_registry ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Service role can read strategy_registry"
  ON strategy_registry FOR SELECT TO service_role USING (true);

CREATE POLICY "Authenticated users can read strategy_registry"
  ON strategy_registry FOR SELECT TO authenticated USING (true);

CREATE POLICY "Service role can manage strategy_registry"
  ON strategy_registry FOR ALL TO service_role USING (true) WITH CHECK (true);

-- Seed data
INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'atrpin',
  'ATR Pin',
  'ATR-based pinbar detection with trend-following entry and exit management',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 14, "required": true, "label": "ATR Window"}, {"key": "multiplier", "type": "number", "default": 10, "label": "ATR Multiplier"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[]'::jsonb,
  10
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'audacitymaker',
  'Audacity Maker',
  'Order flow based market making with per-trade order management',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"orderFlow": {"interval": "1h"}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window", "description": "Indicator window size"}]'::jsonb,
  '[]'::jsonb,
  20
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'autoborrow',
  'Auto Borrow/Repay',
  'Automatically manage margin borrowing and repayment based on margin level',
  'utility',
  '["binance", "okex", "bybit", "bitget"]'::jsonb,
  true,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "text", "default": "5m", "label": "Check Interval", "description": "How often to check margin level"}, {"key": "minMarginLevel", "type": "number", "default": 1.5, "label": "Min Margin Level", "description": "Repay when margin level drops below this"}, {"key": "maxMarginLevel", "type": "number", "default": 3, "label": "Max Margin Level", "description": "Target margin level after repayment"}, {"key": "autoRepayWhenDeposit", "type": "boolean", "default": "true", "label": "Auto Repay on Deposit"}]'::jsonb,
  '[]'::jsonb,
  30
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'autobuy',
  'Auto Buy',
  'Automatic periodic buying with fixed amount',
  'dca',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"bollinger": {"interval": "1h", "window": 20}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "schedule", "type": "text", "default": "0 10 * * *", "required": true, "label": "Schedule", "description": "Cron schedule (e.g., \"0 10 * * *\" for daily 10am)"}, {"key": "quantity", "type": "number", "default": 0.001, "label": "Quantity", "description": "Buy quantity per order"}, {"key": "amount", "type": "number", "default": 100, "label": "Amount", "description": "Buy amount in quote currency (alternative to quantity)"}, {"key": "minBaseBalance", "type": "number", "default": 0, "label": "Min Base Balance", "description": "Only buy if base balance above this"}, {"key": "dryRun", "type": "boolean", "default": "false", "label": "Dry Run", "description": "Simulate without real orders"}]'::jsonb,
  '[]'::jsonb,
  40
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'bollgrid',
  'Bollinger Grid',
  'Grid trading with Bollinger Band dynamic boundaries',
  'grid',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "profitSpread": 50, "quantity": 0.001, "gridNumber": 10, "gridPips": 50}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "gridNumber", "type": "number", "default": 10, "required": true, "label": "Grid Number"}, {"key": "gridPips", "type": "number", "default": 50, "required": true, "label": "Grid Pips", "description": "Grid spacing in pips"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "profitSpread", "type": "number", "default": 50, "required": true, "label": "Profit Spread", "description": "Fixed profit spread (absolute price)"}]'::jsonb,
  '[]'::jsonb,
  50
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'bollmaker',
  'Bollinger Maker',
  'Market making with Bollinger Band neutral zone and trend-following exposure control',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001, "defaultBollinger": {"interval": "1h", "window": 20}, "neutralBollinger": {"interval": "1h", "window": 20}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "bidQuantity", "type": "number", "default": 0.001, "required": true, "label": "Bid Quantity"}, {"key": "askQuantity", "type": "number", "default": 0.001, "required": true, "label": "Ask Quantity"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread", "description": "Price spread from mid price"}, {"key": "minProfitSpread", "type": "number", "default": 0.001, "label": "Min Profit Spread", "description": "Minimum profit spread from average cost"}, {"key": "maxExposurePosition", "type": "number", "default": 1, "label": "Max Exposure", "description": "Maximum position to hold"}, {"key": "disableShort", "type": "boolean", "default": "false", "label": "Disable Short"}, {"key": "tradeInBand", "type": "boolean", "default": "false", "label": "Trade In Band", "description": "Only trade within Bollinger Band"}, {"key": "shadowProtection", "type": "boolean", "default": "false", "label": "Shadow Protection", "description": "Avoid placing orders during strong price drops"}]'::jsonb,
  '[]'::jsonb,
  60
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'convert',
  'Asset Converter',
  'Automatically convert one asset to another (e.g., dust conversion)',
  'utility',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  true,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol", "description": "Trading pair for conversion"}, {"key": "from", "type": "text", "default": "BTC", "required": true, "label": "From Asset", "description": "Source asset to convert"}, {"key": "to", "type": "text", "default": "USDT", "required": true, "label": "To Asset", "description": "Target asset to receive"}]'::jsonb,
  '[]'::jsonb,
  70
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'dca',
  'Dollar Cost Average',
  'Periodic fixed-amount purchases on schedule',
  'dca',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"investmentInterval": "1d", "budget": 500}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "investmentInterval", "type": "select", "default": "1h", "options": ["15m", "1h", "4h", "1d", "1w"], "label": "Investment Interval", "description": "Interval between investments"}, {"key": "budget", "type": "number", "default": 100, "required": true, "label": "Budget Amount", "description": "Budget per period"}, {"key": "budgetPeriod", "type": "select", "default": "day", "options": ["day", "week", "month"], "label": "Budget Period"}]'::jsonb,
  '[]'::jsonb,
  80
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'dca2',
  'DCA v2',
  'Advanced DCA with take-profit, price deviation, and max order controls',
  'dca',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quoteInvestment", "type": "number", "default": 1000, "required": true, "label": "Quote Investment", "description": "Total quote investment"}, {"key": "maxOrderCount", "type": "number", "default": 5, "required": true, "label": "Max Order Count"}, {"key": "priceDeviation", "type": "number", "default": 0.01, "label": "Price Deviation", "description": "Max price deviation for order placement"}, {"key": "takeProfitRatio", "type": "number", "default": 0.05, "label": "Take Profit Ratio", "description": "Take profit ratio"}]'::jsonb,
  '[]'::jsonb,
  90
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'dca3',
  'DCA v3',
  'DCA v2 variant with additional recovery options',
  'dca',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quoteInvestment", "type": "number", "default": 1000, "required": true, "label": "Quote Investment"}, {"key": "maxOrderCount", "type": "number", "default": 5, "required": true, "label": "Max Order Count"}, {"key": "priceDeviation", "type": "number", "default": 0.01, "label": "Price Deviation"}, {"key": "takeProfitRatio", "type": "number", "default": 0.05, "label": "Take Profit Ratio"}]'::jsonb,
  '[]'::jsonb,
  100
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'deposit2transfer',
  'Deposit Transfer',
  'Automatically detect deposits and transfer assets to trading account',
  'utility',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  true,
  false,
  '{}'::jsonb,
  '[{"key": "assets", "type": "text", "default": "USDT,BTC", "required": true, "label": "Assets", "description": "Comma-separated asset list to watch"}, {"key": "interval", "type": "text", "default": "30s", "label": "Check Interval", "description": "How often to check for deposits"}, {"key": "ignoreDust", "type": "boolean", "default": "true", "label": "Ignore Dust", "description": "Skip very small deposits"}]'::jsonb,
  '[]'::jsonb,
  110
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'drift',
  'Drift',
  'Drift MA strategy with linear regression prediction, ATR stop-loss and trailing exits',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"minInterval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 100, "required": true, "label": "Window", "description": "Main indicator window"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "stoploss", "type": "number", "default": 0.02, "label": "Stop Loss", "description": "Stop loss rate"}, {"key": "useStopLoss", "type": "boolean", "default": "true", "label": "Use Stop Loss"}, {"key": "useAtr", "type": "boolean", "default": "false", "label": "Use ATR Stop", "description": "Use ATR for stop-loss calculation"}, {"key": "predictOffset", "type": "number", "default": 10, "label": "Predict Offset", "description": "Lookback length for prediction"}, {"key": "MinInterval", "type": "select", "default": "5m", "options": ["1m", "5m", "15m", "30m"], "label": "Min Interval", "description": "Minimum interval for stop-loss and trailing exits"}, {"key": "atrWindow", "type": "number", "default": 14, "label": "ATR Window"}, {"key": "generateGraph", "type": "boolean", "default": "true", "label": "Generate Graph", "description": "Generate graph on shutdown in backtest"}]'::jsonb,
  '[]'::jsonb,
  120
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'elliottwave',
  'Elliott Wave',
  'Elliott Wave oscillator with ATR stop-loss, Heikin-Ashi and trailing exits',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"minInterval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "stoploss", "type": "number", "default": 0.02, "label": "Stop Loss"}, {"key": "MinInterval", "type": "select", "default": "5m", "options": ["1m", "5m", "15m", "30m"], "label": "Min Interval", "description": "Minimum interval for stop-loss and trailing exits"}, {"key": "windowATR", "type": "number", "default": 14, "label": "ATR Window"}, {"key": "windowQuick", "type": "number", "default": 5, "label": "Quick Window", "description": "Fast EWO window"}, {"key": "windowSlow", "type": "number", "default": 35, "label": "Slow Window", "description": "Slow EWO window"}, {"key": "useHeikinAshi", "type": "boolean", "default": "false", "label": "Heikin Ashi", "description": "Use Heikin-Ashi candles"}, {"key": "drawGraph", "type": "boolean", "default": "true", "label": "Draw Graph"}]'::jsonb,
  '[]'::jsonb,
  130
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'emacross',
  'EMA Cross',
  'Trend following with fast/slow EMA crossover signals',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "fastWindow": 7, "slowWindow": 25}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "fastWindow", "type": "number", "default": 9, "required": true, "label": "Fast EMA Period"}, {"key": "slowWindow", "type": "number", "default": 21, "required": true, "label": "Slow EMA Period"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[]'::jsonb,
  140
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'ewo_dgtrd',
  'EWO Divergence',
  'Elliott Wave Oscillator divergence trading with CCI-Stochastic filters',
  'mean-reversion',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "sigWin", "type": "number", "default": 5, "label": "Signal Window", "description": "Signal window for EWO"}, {"key": "stoploss", "type": "number", "default": 0.02, "label": "Stop Loss"}, {"key": "useHeikinAshi", "type": "boolean", "default": "false", "label": "Heikin Ashi"}, {"key": "cciStochFilterHigh", "type": "number", "default": 100, "label": "CCI Filter High", "description": "CCI Stochastic high filter"}, {"key": "cciStochFilterLow", "type": "number", "label": "CCI Filter Low", "description": "CCI Stochastic low filter"}, {"key": "ewoChangeFilterHigh", "type": "number", "default": 0, "label": "EWO Filter High"}, {"key": "ewoChangeFilterLow", "type": "number", "default": 0, "label": "EWO Filter Low"}]'::jsonb,
  '[]'::jsonb,
  150
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'factorzoo',
  'Factor Zoo',
  'Multi-factor linear strategy with momentum and exit management',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"linear": {"interval": "1h"}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window"}]'::jsonb,
  '[]'::jsonb,
  160
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'fixedmaker',
  'Fixed Maker',
  'Simple market making with fixed spread and quantity',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1m", "quantity": 0.001, "halfSpread": 0.001}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1m", "options": ["1m", "5m", "15m", "1h", "4h"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "halfSpread", "type": "number", "default": 0.001, "required": true, "label": "Half Spread", "description": "Half of the bid-ask spread"}, {"key": "dryRun", "type": "boolean", "default": "false", "label": "Dry Run", "description": "Simulate without real orders"}]'::jsonb,
  '[]'::jsonb,
  170
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'flashcrash',
  'Flash Crash',
  'Buy during flash crashes with percentage-based grid placement',
  'volatility',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1m"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1m", "options": ["1m", "5m", "15m", "1h"], "label": "Interval"}, {"key": "gridNumber", "type": "number", "default": 10, "required": true, "label": "Grid Number"}, {"key": "percentage", "type": "number", "default": 0.01, "required": true, "label": "Percentage", "description": "Percentage of current price for order spacing"}, {"key": "baseQuantity", "type": "number", "default": 0.001, "required": true, "label": "Base Quantity"}]'::jsonb,
  '[]'::jsonb,
  180
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'fmaker',
  'F Maker',
  'Flexible market maker with advanced order management',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "spread": 0.001, "quantity": 0.001}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread"}, {"key": "minProfitSpread", "type": "number", "default": 0.001, "label": "Min Profit Spread"}]'::jsonb,
  '[]'::jsonb,
  190
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'grid',
  'Grid',
  'Classic grid trading with fixed price boundaries',
  'grid',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"side": "both"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol", "description": "Trading pair"}, {"key": "gridNumber", "type": "number", "default": 10, "required": true, "label": "Grid Number", "description": "Number of grid orders"}, {"key": "upperPrice", "type": "number", "default": 70000, "required": true, "label": "Upper Price", "description": "Upper price boundary"}, {"key": "lowerPrice", "type": "number", "default": 50000, "required": true, "label": "Lower Price", "description": "Lower price boundary"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity", "description": "Quantity per grid order"}, {"key": "profitSpread", "type": "number", "default": 50, "required": true, "label": "Profit Spread", "description": "Fixed profit spread per grid (absolute price difference)"}, {"key": "side", "type": "select", "default": "both", "options": ["buy", "sell", "both"], "label": "Initial Side", "description": "Initial maker orders side"}, {"key": "catchUp", "type": "boolean", "default": "false", "label": "Catch Up", "description": "Enable grid to catch up with price changes"}]'::jsonb,
  '[]'::jsonb,
  200
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'grid2',
  'Grid v2',
  'Advanced grid trading with compound, stop-loss, take-profit and auto-range',
  'grid',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol", "description": "Trading pair"}, {"key": "gridNumber", "type": "number", "default": 10, "required": true, "label": "Grid Number", "description": "Number of grid orders"}, {"key": "upperPrice", "type": "number", "default": 70000, "required": true, "label": "Upper Price", "description": "Upper price boundary (adjust for selected symbol)"}, {"key": "lowerPrice", "type": "number", "default": 50000, "required": true, "label": "Lower Price", "description": "Lower price boundary (adjust for selected symbol)"}, {"key": "quantity", "type": "number", "default": 0.001, "label": "Quantity", "description": "Quantity per grid order (mutually exclusive with Quote Investment)"}, {"key": "profitSpread", "type": "number", "default": 0, "label": "Profit Spread", "description": "Fixed profit spread (absolute price, 0 = auto from grid range)"}, {"key": "quoteInvestment", "type": "number", "default": 1000, "label": "Quote Investment", "description": "Total quote investment amount (overrides Quantity if set)"}, {"key": "compound", "type": "boolean", "default": "false", "label": "Compound", "description": "Reinvest profits"}, {"key": "earnBase", "type": "boolean", "default": "false", "label": "Earn Base", "description": "Earn profit in base currency instead of quote"}, {"key": "triggerPrice", "type": "number", "default": 0, "label": "Trigger Price", "description": "Price to trigger grid opening (0 = disabled)"}, {"key": "stopLossPrice", "type": "number", "default": 0, "label": "Stop Loss Price", "description": "Stop loss price (0 = disabled)"}, {"key": "takeProfitPrice", "type": "number", "default": 0, "label": "Take Profit Price", "description": "Take profit price (0 = disabled)"}]'::jsonb,
  '[]'::jsonb,
  210
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'harmonic',
  'Harmonic',
  'SHARK harmonic pattern detection with quantity-based entries',
  'mean-reversion',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "drawGraph", "type": "boolean", "default": "true", "label": "Draw Graph"}]'::jsonb,
  '[]'::jsonb,
  220
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'irr',
  'Negative Return Rate',
  'Mean reversion using negative return rate indicator with quantity-based entries',
  'mean-reversion',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "drawGraph", "type": "boolean", "default": "true", "label": "Draw Graph"}]'::jsonb,
  '[]'::jsonb,
  230
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'linregmaker',
  'Linear Regression Maker',
  'Market making with linear regression trend indicators for dynamic spread and exposure',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001, "reverseEMA": {"interval": "1h", "window": 100}, "fastLinReg": {"interval": "1h", "window": 30}, "slowLinReg": {"interval": "1h", "window": 60}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "bidQuantity", "type": "number", "default": 0.001, "required": true, "label": "Bid Quantity"}, {"key": "askQuantity", "type": "number", "default": 0.001, "required": true, "label": "Ask Quantity"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread"}, {"key": "minProfitSpread", "type": "number", "default": 0.001, "label": "Min Profit Spread"}, {"key": "maxExposurePosition", "type": "number", "default": 1, "label": "Max Exposure"}]'::jsonb,
  '[]'::jsonb,
  240
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'liquiditymaker',
  'Liquidity Maker',
  'Advanced market maker with layered liquidity and mid-price EMA tracking',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"adjustmentUpdateInterval": "1h", "liquidityUpdateInterval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "numOfLiquidityLayers", "type": "number", "default": 5, "required": true, "label": "Liquidity Layers", "description": "Number of liquidity layers"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread"}, {"key": "askLiquidityAmount", "type": "number", "default": 0.001, "label": "Ask Amount"}, {"key": "bidLiquidityAmount", "type": "number", "default": 0.001, "label": "Bid Amount"}, {"key": "liquidityPriceRange", "type": "number", "default": 0.01, "label": "Price Range", "description": "Liquidity price range ratio"}, {"key": "maxPositionExposure", "type": "number", "default": 1, "label": "Max Exposure"}]'::jsonb,
  '[]'::jsonb,
  250
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'pivotshort',
  'Pivot Short',
  'Short trades based on pivot point breakouts with RSI filter',
  'mean-reversion',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "leverage", "type": "number", "default": 1, "label": "Leverage"}]'::jsonb,
  '[]'::jsonb,
  260
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'random',
  'Random',
  'Random trading for testing and benchmarking',
  'other',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"schedule": "*/30 * * * *"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quantity", "type": "number", "default": 0.001, "label": "Quantity"}, {"key": "schedule", "type": "text", "default": "*/30 * * * *", "label": "Cron Schedule", "description": "Cron expression for trade timing"}, {"key": "dryRun", "type": "boolean", "default": "true", "label": "Dry Run", "description": "Simulate without real orders"}]'::jsonb,
  '[]'::jsonb,
  270
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'rebalance',
  'Rebalance',
  'Periodic portfolio rebalancing across assets',
  'other',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"schedule": "0 */4 * * *", "quoteCurrency": "USDT"}'::jsonb,
  '[{"key": "schedule", "type": "text", "default": "0 0 * * *", "required": true, "label": "Schedule", "description": "Cron schedule (e.g., \"0 0 * * *\" for daily)"}, {"key": "quoteCurrency", "type": "text", "default": "USDT", "required": true, "label": "Quote Currency"}, {"key": "threshold", "type": "number", "default": 0.05, "label": "Rebalance Threshold", "description": "Deviation threshold to trigger rebalance"}, {"key": "dryRun", "type": "boolean", "default": "false", "label": "Dry Run", "description": "Simulate without real orders"}]'::jsonb,
  '[]'::jsonb,
  280
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'rsmaker',
  'RS Maker',
  'Market making with Relative Strength indicator for trend-aware order placement',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "bidQuantity": 0.001, "askQuantity": 0.001, "neutralBollinger": {"interval": "1h", "window": 20, "bandWidth": 2.0}, "defaultBollinger": {"interval": "1h", "window": 20, "bandWidth": 2.0}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "bidQuantity", "type": "number", "default": 0.001, "required": true, "label": "Bid Quantity"}, {"key": "askQuantity", "type": "number", "default": 0.001, "required": true, "label": "Ask Quantity"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread"}, {"key": "minProfitSpread", "type": "number", "default": 0.001, "label": "Min Profit Spread"}, {"key": "maxExposurePosition", "type": "number", "default": 1, "label": "Max Exposure"}]'::jsonb,
  '[]'::jsonb,
  290
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'schedule',
  'Scheduled Order',
  'Submit orders on schedule with optional moving average conditions',
  'dca',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["15m", "1h", "4h", "1d", "1w"], "label": "Interval"}, {"key": "side", "type": "select", "default": "buy", "required": true, "options": ["buy", "sell"], "label": "Side"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "useLimitOrder", "type": "boolean", "default": "false", "label": "Limit Order", "description": "Use limit orders instead of market"}, {"key": "minBaseBalance", "type": "number", "default": 0, "label": "Min Base Balance", "description": "Minimum base balance to place sell orders"}, {"key": "maxBaseBalance", "type": "number", "default": 0, "label": "Max Base Balance", "description": "Maximum base balance for buy orders (0 = disabled)"}]'::jsonb,
  '[]'::jsonb,
  300
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'scmaker',
  'SC Maker',
  'Market making with Bollinger Band safety and grid scaling',
  'maker',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"adjustmentUpdateInterval": "1h", "liquidityUpdateInterval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "window", "type": "number", "default": 20, "required": true, "label": "Window", "description": "Indicator window size"}, {"key": "k", "type": "number", "default": 0.5, "label": "K Factor", "description": "K factor for strength calculation"}, {"key": "numOfLiquidityLayers", "type": "number", "default": 5, "required": true, "label": "Liquidity Layers"}, {"key": "maxExposure", "type": "number", "default": 1, "label": "Max Exposure"}, {"key": "minProfit", "type": "number", "default": 0.001, "label": "Min Profit"}]'::jsonb,
  '[]'::jsonb,
  310
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'sentinel',
  'Sentinel',
  'Monitor and alert on price movements and market conditions',
  'other',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  true,
  false,
  '{"interval": "1m"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "threshold", "type": "number", "default": 0.5, "label": "Threshold", "description": "Sensitivity for anomaly detection"}, {"key": "window", "type": "number", "default": 100, "label": "Window", "description": "Lookback window for detection"}, {"key": "numSamples", "type": "number", "default": 50, "label": "Samples", "description": "Number of samples for training"}, {"key": "proportion", "type": "number", "default": 0.1, "label": "Proportion", "description": "Proportion of outliers expected"}]'::jsonb,
  '[]'::jsonb,
  320
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'supertrend',
  'Supertrend',
  'Trend following with Supertrend indicator and optional DEMA confirmation',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "supertrendMultiplier", "type": "number", "default": 3, "label": "ATR Multiplier", "description": "ATR multiplier for supertrend"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "leverage", "type": "number", "default": 1, "label": "Leverage"}, {"key": "takeProfitAtrMultiplier", "type": "number", "default": 2, "label": "Take Profit ATR Mult", "description": "Take profit as multiple of ATR"}, {"key": "stopByReversedSupertrend", "type": "boolean", "default": "false", "label": "Stop on Reversal", "description": "Exit when supertrend signal reverses"}, {"key": "drawGraph", "type": "boolean", "default": "true", "label": "Draw Graph", "description": "Draw PNL graph in backtest"}]'::jsonb,
  '[]'::jsonb,
  330
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'support',
  'Support Monitor',
  'Detect support/resistance levels and trigger protective orders',
  'utility',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "quantity": 0.001}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[]'::jsonb,
  340
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'swing',
  'Swing',
  'Swing trading with moving average crossover and minimum change filter',
  'mean-reversion',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "movingAverageType": "SMA", "movingAverageWindow": 20, "movingAverageInterval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "baseQuantity", "type": "number", "default": 0.001, "required": true, "label": "Base Quantity"}, {"key": "minChange", "type": "number", "default": 0.01, "label": "Min Change", "description": "Minimum price change to trigger trade"}, {"key": "movingAverageType", "type": "select", "default": "SMA", "options": ["SMA", "EWMA"], "label": "MA Type"}, {"key": "movingAverageInterval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "MA Interval", "description": "Interval for moving average calculation"}, {"key": "movingAverageWindow", "type": "number", "default": 20, "label": "MA Window"}]'::jsonb,
  '[]'::jsonb,
  350
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'techsignal',
  'Tech Signal',
  'Technical signal detection with support levels and funding rate monitoring',
  'indicator',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}]'::jsonb,
  '[]'::jsonb,
  360
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'trendtrader',
  'Trend Trader',
  'Trend line breakout trading with configurable entry methods',
  'trend',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{"interval": "1h", "trend": {"interval": "1h"}}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["1m", "5m", "15m", "1h", "4h", "1d"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "leverage", "type": "number", "default": 1, "label": "Leverage"}]'::jsonb,
  '[]'::jsonb,
  370
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'wall',
  'Wall',
  'Place large wall orders at configurable levels',
  'other',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "side", "type": "select", "default": "sell", "required": true, "options": ["buy", "sell"], "label": "Side"}, {"key": "interval", "type": "select", "default": "1m", "options": ["1m", "5m", "15m", "1h"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.1, "required": true, "label": "Quantity"}, {"key": "numLayers", "type": "number", "default": 5, "required": true, "label": "Num Layers"}, {"key": "layerSpread", "type": "number", "default": 0.01, "label": "Layer Spread", "description": "Spread between each layer"}]'::jsonb,
  '[]'::jsonb,
  380
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xalign',
  'Cross-Exchange Align',
  'Align positions across spot and futures sessions with arbitrage logic',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h"], "label": "Interval"}]'::jsonb,
  '[{"name": "spot", "label": "Spot Session", "futures": false}, {"name": "futures", "label": "Futures Session", "futures": true}]'::jsonb,
  390
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xbalance',
  'Cross-Exchange Balance',
  'Balance rebalancing across two exchange sessions',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h", "1d"], "label": "Interval"}]'::jsonb,
  '[{"name": "source", "label": "Source Session", "futures": false}, {"name": "target", "label": "Target Session", "futures": false}]'::jsonb,
  400
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xdepthmaker',
  'Cross-Exchange Depth Maker',
  'Depth-based market maker with cross-exchange order book comparison',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread"}]'::jsonb,
  '[{"name": "maker", "label": "Maker Session", "futures": false}, {"name": "hedge", "label": "Hedge Session", "futures": true}]'::jsonb,
  410
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xfixedmaker',
  'Cross-Exchange Fixed Maker',
  'Fixed spread market maker with cross-exchange hedging',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "halfSpread", "type": "number", "default": 0.001, "label": "Half Spread", "description": "Half of the bid-ask spread"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[{"name": "maker", "label": "Maker Session", "futures": false}, {"name": "hedge", "label": "Hedge Session", "futures": true}]'::jsonb,
  420
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xfunding',
  'Funding Rate Arbitrage',
  'Capture funding rate differentials between spot and perpetual futures',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[{"name": "spot", "label": "Spot Session", "futures": false}, {"name": "futures", "label": "Futures Session", "futures": true}]'::jsonb,
  430
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xfundingv2',
  'Funding Rate Arbitrage v2',
  'Enhanced funding rate arbitrage with improved position management',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[{"name": "spot", "label": "Spot Session", "futures": false}, {"name": "futures", "label": "Futures Session", "futures": true}]'::jsonb,
  440
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xgap',
  'Cross-Exchange Gap',
  'Detect and trade price gaps across two exchanges',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[{"name": "sessionA", "label": "Session A", "futures": false}, {"name": "sessionB", "label": "Session B", "futures": false}]'::jsonb,
  450
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xhedgegrid',
  'Hedge Grid',
  'Grid trading with hedge mode support and compound/profit options',
  'grid',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "gridNumber", "type": "number", "default": 10, "required": true, "label": "Grid Number"}, {"key": "upperPrice", "type": "number", "default": 70000, "required": true, "label": "Upper Price", "description": "Upper price boundary (adjust for selected symbol)"}, {"key": "lowerPrice", "type": "number", "default": 50000, "required": true, "label": "Lower Price", "description": "Lower price boundary (adjust for selected symbol)"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "profitSpread", "type": "number", "default": 0, "label": "Profit Spread", "description": "Fixed profit spread (absolute price, 0 = auto)"}, {"key": "quoteInvestment", "type": "number", "default": 1000, "label": "Quote Investment"}, {"key": "compound", "type": "boolean", "default": "false", "label": "Compound", "description": "Reinvest profits"}, {"key": "earnBase", "type": "boolean", "default": "false", "label": "Earn Base", "description": "Earn profit in base currency"}]'::jsonb,
  '[]'::jsonb,
  460
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xmaker',
  'Cross-Exchange Market Maker',
  'Market maker that hedges orders across two exchanges (maker + hedge sessions)',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{"quantity": 0.001}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "spread", "type": "number", "default": 0.001, "label": "Spread", "description": "Bid-ask spread ratio"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "updateInterval", "type": "text", "default": "1m", "label": "Update Interval", "description": "Order book update interval"}, {"key": "hedgeInterval", "type": "text", "default": "5m", "label": "Hedge Interval", "description": "Hedge order check interval"}]'::jsonb,
  '[{"name": "maker", "label": "Maker Session", "futures": false}, {"name": "hedge", "label": "Hedge Session", "futures": true}]'::jsonb,
  470
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xnav',
  'Cross-Exchange NAV',
  'Net asset value tracking and rebalancing across sessions',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{"interval": "1h"}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "1h", "options": ["5m", "15m", "1h", "4h"], "label": "Interval"}]'::jsonb,
  '[{"name": "spot", "label": "Spot Session", "futures": false}, {"name": "futures", "label": "Futures Session", "futures": true}]'::jsonb,
  480
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xpremium',
  'Cross-Exchange Premium',
  'Trade premium/discount between spot and futures across exchanges',
  'cross-exchange',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  true,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "interval", "type": "select", "default": "5m", "options": ["1m", "5m", "15m", "1h"], "label": "Interval"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}]'::jsonb,
  '[{"name": "spot", "label": "Spot Session", "futures": false}, {"name": "futures", "label": "Futures Session", "futures": true}]'::jsonb,
  490
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

INSERT INTO strategy_registry (id, display_name, description, category, exchanges, live_only, cross_exchange, defaults, fields, session_roles, sort_order)
VALUES (
  'xvs',
  'Volume Surge',
  'Trade on volume surge signals with EMA and pivot high confirmation',
  'volatility',
  '["binance", "okex", "bybit", "bitget", "kucoin"]'::jsonb,
  false,
  false,
  '{}'::jsonb,
  '[{"key": "symbol", "type": "text", "default": "BTCUSDT", "required": true, "label": "Symbol"}, {"key": "quantity", "type": "number", "default": 0.001, "required": true, "label": "Quantity"}, {"key": "maxExposure", "type": "number", "default": 1, "label": "Max Exposure"}, {"key": "volumeInterval", "type": "select", "default": "5m", "options": ["1m", "5m", "15m", "1h"], "label": "Volume Interval"}, {"key": "volumeThreshold", "type": "number", "default": 800, "label": "Volume Threshold", "description": "Base asset volume threshold"}, {"key": "stoploss", "type": "number", "default": 0.02, "label": "Stop Loss"}]'::jsonb,
  '[]'::jsonb,
  500
) ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  category = EXCLUDED.category,
  exchanges = EXCLUDED.exchanges,
  live_only = EXCLUDED.live_only,
  cross_exchange = EXCLUDED.cross_exchange,
  defaults = EXCLUDED.defaults,
  fields = EXCLUDED.fields,
  session_roles = EXCLUDED.session_roles,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_updated_at ON strategy_registry;
CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON strategy_registry
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();