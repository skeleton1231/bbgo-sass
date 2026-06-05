export type FieldType = 'number' | 'text' | 'boolean' | 'select' | 'group'

export interface FieldDef {
  key: string
  label: string
  type: FieldType
  default: string | number | boolean
  options?: string[]
  min?: number
  max?: number
  step?: number
  required?: boolean
  description?: string
}

export interface SessionRole {
  name: string
  label: string
  futures: boolean
}

export interface StrategySchema {
  id: string
  label: string
  description: string
  category: 'grid' | 'maker' | 'trend' | 'mean-reversion' | 'dca' | 'volatility' | 'other' | 'indicator' | 'cross-exchange' | 'utility'
  supportedExchanges: string[]
  fields: FieldDef[]
  crossExchange?: boolean
  sessionRoles?: SessionRole[]
  liveOnly?: boolean
}

const STRATEGY_SCHEMAS: StrategySchema[] = [
  // ===================== Grid Strategies =====================
  {
    id: 'grid',
    label: 'Grid',
    description: 'Classic grid trading with fixed price boundaries',
    category: 'grid',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true, description: 'Trading pair' },
      { key: 'gridNumber', label: 'Grid Number', type: 'number', default: 10, min: 2, max: 200, required: true, description: 'Number of grid orders' },
      { key: 'upperPrice', label: 'Upper Price', type: 'number', default: 70000, step: 0.01, required: true, description: 'Upper price boundary' },
      { key: 'lowerPrice', label: 'Lower Price', type: 'number', default: 50000, step: 0.01, required: true, description: 'Lower price boundary' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true, description: 'Quantity per grid order' },
      { key: 'profitSpread', label: 'Profit Spread', type: 'number', default: 50, step: 0.01, required: true, description: 'Fixed profit spread per grid (absolute price difference)' },
      { key: 'side', label: 'Initial Side', type: 'select', default: 'both', options: ['buy', 'sell', 'both'], description: 'Initial maker orders side' },
      { key: 'catchUp', label: 'Catch Up', type: 'boolean', default: false, description: 'Enable grid to catch up with price changes' },
    ],
  },
  {
    id: 'grid2',
    label: 'Grid v2',
    description: 'Advanced grid trading with compound, stop-loss, take-profit and auto-range',
    category: 'grid',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true, description: 'Trading pair' },
      { key: 'gridNumber', label: 'Grid Number', type: 'number', default: 10, min: 2, max: 200, required: true, description: 'Number of grid orders' },
      { key: 'upperPrice', label: 'Upper Price', type: 'number', default: 70000, step: 0.01, required: true, description: 'Upper price boundary (adjust for selected symbol)' },
      { key: 'lowerPrice', label: 'Lower Price', type: 'number', default: 50000, step: 0.01, required: true, description: 'Lower price boundary (adjust for selected symbol)' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, description: 'Quantity per grid order (mutually exclusive with Quote Investment)' },
      { key: 'profitSpread', label: 'Profit Spread', type: 'number', default: 0, step: 0.01, description: 'Fixed profit spread (absolute price, 0 = auto from grid range)' },
      { key: 'quoteInvestment', label: 'Quote Investment', type: 'number', default: 1000, step: 0.01, description: 'Total quote investment amount (overrides Quantity if set)' },
      { key: 'compound', label: 'Compound', type: 'boolean', default: false, description: 'Reinvest profits' },
      { key: 'earnBase', label: 'Earn Base', type: 'boolean', default: false, description: 'Earn profit in base currency instead of quote' },
      { key: 'triggerPrice', label: 'Trigger Price', type: 'number', default: 0, step: 0.01, description: 'Price to trigger grid opening (0 = disabled)' },
      { key: 'stopLossPrice', label: 'Stop Loss Price', type: 'number', default: 0, step: 0.01, description: 'Stop loss price (0 = disabled)' },
      { key: 'takeProfitPrice', label: 'Take Profit Price', type: 'number', default: 0, step: 0.01, description: 'Take profit price (0 = disabled)' },
    ],
  },
  {
    id: 'bollgrid',
    label: 'Bollinger Grid',
    description: 'Grid trading with Bollinger Band dynamic boundaries',
    category: 'grid',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'gridNumber', label: 'Grid Number', type: 'number', default: 10, min: 2, max: 100, required: true },
      { key: 'gridPips', label: 'Grid Pips', type: 'number', default: 50, step: 0.01, required: true, description: 'Grid spacing in pips' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'profitSpread', label: 'Profit Spread', type: 'number', default: 50, step: 0.01, required: true, description: 'Fixed profit spread (absolute price)' },
    ],
  },

  // ===================== Market Maker Strategies =====================
  {
    id: 'bollmaker',
    label: 'Bollinger Maker',
    description: 'Market making with Bollinger Band neutral zone and trend-following exposure control',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'bidQuantity', label: 'Bid Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'askQuantity', label: 'Ask Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001, description: 'Price spread from mid price' },
      { key: 'minProfitSpread', label: 'Min Profit Spread', type: 'number', default: 0.001, step: 0.0001, description: 'Minimum profit spread from average cost' },
      { key: 'maxExposurePosition', label: 'Max Exposure', type: 'number', default: 1, step: 0.01, description: 'Maximum position to hold' },
      { key: 'disableShort', label: 'Disable Short', type: 'boolean', default: false },
      { key: 'tradeInBand', label: 'Trade In Band', type: 'boolean', default: false, description: 'Only trade within Bollinger Band' },
      { key: 'shadowProtection', label: 'Shadow Protection', type: 'boolean', default: false, description: 'Avoid placing orders during strong price drops' },
    ],
  },
  {
    id: 'linregmaker',
    label: 'Linear Regression Maker',
    description: 'Market making with linear regression trend indicators for dynamic spread and exposure',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'bidQuantity', label: 'Bid Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'askQuantity', label: 'Ask Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'minProfitSpread', label: 'Min Profit Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'maxExposurePosition', label: 'Max Exposure', type: 'number', default: 1, step: 0.01 },
    ],
  },
  {
    id: 'rsmaker',
    label: 'RS Maker',
    description: 'Market making with Relative Strength indicator for trend-aware order placement',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'bidQuantity', label: 'Bid Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'askQuantity', label: 'Ask Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'minProfitSpread', label: 'Min Profit Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'maxExposurePosition', label: 'Max Exposure', type: 'number', default: 1, step: 0.01 },
    ],
  },
  {
    id: 'fixedmaker',
    label: 'Fixed Maker',
    description: 'Simple market making with fixed spread and quantity',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1m', options: ['1m', '5m', '15m', '1h', '4h'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'halfSpread', label: 'Half Spread', type: 'number', default: 0.001, step: 0.0001, required: true, description: 'Half of the bid-ask spread' },
      { key: 'dryRun', label: 'Dry Run', type: 'boolean', default: false, description: 'Simulate without real orders' },
    ],
  },
  {
    id: 'fmaker',
    label: 'F Maker',
    description: 'Flexible market maker with advanced order management',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'minProfitSpread', label: 'Min Profit Spread', type: 'number', default: 0.001, step: 0.0001 },
    ],
  },
  {
    id: 'scmaker',
    label: 'SC Maker',
    description: 'Market making with Bollinger Band safety and grid scaling',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'Window', type: 'number', default: 20, min: 1, max: 500, required: true, description: 'Indicator window size' },
      { key: 'k', label: 'K Factor', type: 'number', default: 0.5, step: 0.1, description: 'K factor for strength calculation' },
      { key: 'numOfLiquidityLayers', label: 'Liquidity Layers', type: 'number', default: 5, min: 1, max: 50, required: true },
      { key: 'maxExposure', label: 'Max Exposure', type: 'number', default: 1, step: 0.01 },
      { key: 'minProfit', label: 'Min Profit', type: 'number', default: 0.001, step: 0.0001 },
    ],
  },

  // ===================== Trend Following Strategies =====================
  {
    id: 'supertrend',
    label: 'Supertrend',
    description: 'Trend following with Supertrend indicator and optional DEMA confirmation',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'supertrendMultiplier', label: 'ATR Multiplier', type: 'number', default: 3, step: 0.1, min: 0.1, description: 'ATR multiplier for supertrend' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'leverage', label: 'Leverage', type: 'number', default: 1, min: 1, max: 125 },
      { key: 'takeProfitAtrMultiplier', label: 'Take Profit ATR Mult', type: 'number', default: 2, step: 0.1, description: 'Take profit as multiple of ATR' },
      { key: 'stopByReversedSupertrend', label: 'Stop on Reversal', type: 'boolean', default: false, description: 'Exit when supertrend signal reverses' },
      { key: 'drawGraph', label: 'Draw Graph', type: 'boolean', default: true, description: 'Draw PNL graph in backtest' },
    ],
  },
  {
    id: 'emacross',
    label: 'EMA Cross',
    description: 'Trend following with fast/slow EMA crossover signals',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'fastWindow', label: 'Fast EMA Period', type: 'number', default: 9, min: 1, max: 200, required: true },
      { key: 'slowWindow', label: 'Slow EMA Period', type: 'number', default: 21, min: 1, max: 200, required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'trendtrader',
    label: 'Trend Trader',
    description: 'Trend line breakout trading with configurable entry methods',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'leverage', label: 'Leverage', type: 'number', default: 1, min: 1, max: 125 },
    ],
  },

  // ===================== Mean Reversion Strategies =====================
  {
    id: 'pivotshort',
    label: 'Pivot Short',
    description: 'Short trades based on pivot point breakouts with RSI filter',
    category: 'mean-reversion',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'leverage', label: 'Leverage', type: 'number', default: 1, min: 1, max: 125 },
    ],
  },
  {
    id: 'swing',
    label: 'Swing',
    description: 'Swing trading with moving average crossover and minimum change filter',
    category: 'mean-reversion',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'baseQuantity', label: 'Base Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'minChange', label: 'Min Change', type: 'number', default: 0.01, step: 0.001, description: 'Minimum price change to trigger trade' },
      { key: 'movingAverageType', label: 'MA Type', type: 'select', default: 'SMA', options: ['SMA', 'EWMA'] },
      { key: 'movingAverageInterval', label: 'MA Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'], description: 'Interval for moving average calculation' },
      { key: 'movingAverageWindow', label: 'MA Window', type: 'number', default: 20, min: 1, max: 500 },
    ],
  },

  // ===================== DCA Strategies =====================
  {
    id: 'dca',
    label: 'Dollar Cost Average',
    description: 'Periodic fixed-amount purchases on schedule',
    category: 'dca',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'investmentInterval', label: 'Investment Interval', type: 'select', default: '1h', options: ['15m', '1h', '4h', '1d', '1w'], description: 'Interval between investments' },
      { key: 'budget', label: 'Budget Amount', type: 'number', default: 100, step: 0.01, required: true, description: 'Budget per period' },
      { key: 'budgetPeriod', label: 'Budget Period', type: 'select', default: 'day', options: ['day', 'week', 'month'] },
    ],
  },
  {
    id: 'dca2',
    label: 'DCA v2',
    description: 'Advanced DCA with take-profit, price deviation, and max order controls',
    category: 'dca',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quoteInvestment', label: 'Quote Investment', type: 'number', default: 1000, step: 0.01, required: true, description: 'Total quote investment' },
      { key: 'maxOrderCount', label: 'Max Order Count', type: 'number', default: 5, min: 1, max: 50, required: true },
      { key: 'priceDeviation', label: 'Price Deviation', type: 'number', default: 0.01, step: 0.001, description: 'Max price deviation for order placement' },
      { key: 'takeProfitRatio', label: 'Take Profit Ratio', type: 'number', default: 0.05, step: 0.001, description: 'Take profit ratio' },
    ],
  },
  {
    id: 'dca3',
    label: 'DCA v3',
    description: 'DCA v2 variant with additional recovery options',
    category: 'dca',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quoteInvestment', label: 'Quote Investment', type: 'number', default: 1000, step: 0.01, required: true },
      { key: 'maxOrderCount', label: 'Max Order Count', type: 'number', default: 5, min: 1, max: 50, required: true },
      { key: 'priceDeviation', label: 'Price Deviation', type: 'number', default: 0.01, step: 0.001 },
      { key: 'takeProfitRatio', label: 'Take Profit Ratio', type: 'number', default: 0.05, step: 0.001 },
    ],
  },
  {
    id: 'autobuy',
    label: 'Auto Buy',
    description: 'Automatic periodic buying with fixed amount',
    category: 'dca',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'schedule', label: 'Schedule', type: 'text', default: '0 10 * * *', required: true, description: 'Cron schedule (e.g., "0 10 * * *" for daily 10am)' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, description: 'Buy quantity per order' },
      { key: 'amount', label: 'Amount', type: 'number', default: 100, step: 0.01, description: 'Buy amount in quote currency (alternative to quantity)' },
      { key: 'minBaseBalance', label: 'Min Base Balance', type: 'number', default: 0, step: 0.01, description: 'Only buy if base balance above this' },
      { key: 'dryRun', label: 'Dry Run', type: 'boolean', default: false, description: 'Simulate without real orders' },
    ],
  },

  // ===================== Volatility Strategies =====================
  {
    id: 'flashcrash',
    label: 'Flash Crash',
    description: 'Buy during flash crashes with percentage-based grid placement',
    category: 'volatility',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1m', options: ['1m', '5m', '15m', '1h'] },
      { key: 'gridNumber', label: 'Grid Number', type: 'number', default: 10, min: 1, max: 100, required: true },
      { key: 'percentage', label: 'Percentage', type: 'number', default: 0.01, step: 0.001, required: true, description: 'Percentage of current price for order spacing' },
      { key: 'baseQuantity', label: 'Base Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },

  // ===================== Other Strategies =====================
  {
    id: 'wall',
    label: 'Wall',
    description: 'Place large wall orders at configurable levels',
    category: 'other',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'side', label: 'Side', type: 'select', default: 'sell', options: ['buy', 'sell'], required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1m', options: ['1m', '5m', '15m', '1h'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.1, step: 0.001, required: true },
      { key: 'numLayers', label: 'Num Layers', type: 'number', default: 5, min: 1, max: 50, required: true },
      { key: 'layerSpread', label: 'Layer Spread', type: 'number', default: 0.01, step: 0.001, description: 'Spread between each layer' },
    ],
  },
  {
    id: 'sentinel',
    label: 'Sentinel',
    description: 'Monitor and alert on price movements and market conditions',
    category: 'other',
    liveOnly: true,
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'threshold', label: 'Threshold', type: 'number', default: 0.5, step: 0.1, description: 'Sensitivity for anomaly detection' },
      { key: 'window', label: 'Window', type: 'number', default: 100, min: 10, max: 1000, description: 'Lookback window for detection' },
      { key: 'numSamples', label: 'Samples', type: 'number', default: 50, min: 10, description: 'Number of samples for training' },
      { key: 'proportion', label: 'Proportion', type: 'number', default: 0.1, step: 0.01, description: 'Proportion of outliers expected' },
    ],
  },
  {
    id: 'random',
    label: 'Random',
    description: 'Random trading for testing and benchmarking',
    category: 'other',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'schedule', label: 'Cron Schedule', type: 'text', default: '*/30 * * * *', description: 'Cron expression for trade timing' },
      { key: 'dryRun', label: 'Dry Run', type: 'boolean', default: true, description: 'Simulate without real orders' },
    ],
  },
  {
    id: 'rebalance',
    label: 'Rebalance',
    description: 'Periodic portfolio rebalancing across assets',
    category: 'other',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'schedule', label: 'Schedule', type: 'text', default: '0 0 * * *', required: true, description: 'Cron schedule (e.g., "0 0 * * *" for daily)' },
      { key: 'quoteCurrency', label: 'Quote Currency', type: 'text', default: 'USDT', required: true },
      { key: 'threshold', label: 'Rebalance Threshold', type: 'number', default: 0.05, step: 0.01, description: 'Deviation threshold to trigger rebalance' },
      { key: 'dryRun', label: 'Dry Run', type: 'boolean', default: false, description: 'Simulate without real orders' },
    ],
  },

  // ===================== Additional Grid Strategies =====================
  {
    id: 'xhedgegrid',
    label: 'Hedge Grid',
    description: 'Grid trading with hedge mode support and compound/profit options',
    category: 'grid',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'gridNumber', label: 'Grid Number', type: 'number', default: 10, min: 2, max: 200, required: true },
      { key: 'upperPrice', label: 'Upper Price', type: 'number', default: 70000, step: 0.01, required: true, description: 'Upper price boundary (adjust for selected symbol)' },
      { key: 'lowerPrice', label: 'Lower Price', type: 'number', default: 50000, step: 0.01, required: true, description: 'Lower price boundary (adjust for selected symbol)' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'profitSpread', label: 'Profit Spread', type: 'number', default: 0, step: 0.01, description: 'Fixed profit spread (absolute price, 0 = auto)' },
      { key: 'quoteInvestment', label: 'Quote Investment', type: 'number', default: 1000, step: 0.01 },
      { key: 'compound', label: 'Compound', type: 'boolean', default: false, description: 'Reinvest profits' },
      { key: 'earnBase', label: 'Earn Base', type: 'boolean', default: false, description: 'Earn profit in base currency' },
    ],
  },

  // ===================== Additional Market Maker Strategies =====================
  {
    id: 'audacitymaker',
    label: 'Audacity Maker',
    description: 'Order flow based market making with per-trade order management',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'Window', type: 'number', default: 20, min: 1, max: 500, required: true, description: 'Indicator window size' },
    ],
  },
  {
    id: 'liquiditymaker',
    label: 'Liquidity Maker',
    description: 'Advanced market maker with layered liquidity and mid-price EMA tracking',
    category: 'maker',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'numOfLiquidityLayers', label: 'Liquidity Layers', type: 'number', default: 5, min: 1, max: 50, required: true, description: 'Number of liquidity layers' },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'askLiquidityAmount', label: 'Ask Amount', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'bidLiquidityAmount', label: 'Bid Amount', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'liquidityPriceRange', label: 'Price Range', type: 'number', default: 0.01, step: 0.001, description: 'Liquidity price range ratio' },
      { key: 'maxPositionExposure', label: 'Max Exposure', type: 'number', default: 1, step: 0.01 },
    ],
  },

  // ===================== Additional Trend Strategies =====================
  {
    id: 'atrpin',
    label: 'ATR Pin',
    description: 'ATR-based pinbar detection with trend-following entry and exit management',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'ATR Window', type: 'number', default: 14, min: 1, max: 200, required: true },
      { key: 'multiplier', label: 'ATR Multiplier', type: 'number', default: 10, step: 0.1, min: 0.1 },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'drift',
    label: 'Drift',
    description: 'Drift MA strategy with linear regression prediction, ATR stop-loss and trailing exits',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'Window', type: 'number', default: 100, min: 5, max: 500, required: true, description: 'Main indicator window' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'stoploss', label: 'Stop Loss', type: 'number', default: 0.02, step: 0.001, description: 'Stop loss rate' },
      { key: 'useStopLoss', label: 'Use Stop Loss', type: 'boolean', default: true },
      { key: 'useAtr', label: 'Use ATR Stop', type: 'boolean', default: false, description: 'Use ATR for stop-loss calculation' },
      { key: 'predictOffset', label: 'Predict Offset', type: 'number', default: 10, min: 1, description: 'Lookback length for prediction' },
      { key: 'MinInterval', label: 'Min Interval', type: 'select', default: '5m', options: ['1m', '5m', '15m', '30m'], description: 'Minimum interval for stop-loss and trailing exits' },
      { key: 'atrWindow', label: 'ATR Window', type: 'number', default: 14, min: 1 },
      { key: 'generateGraph', label: 'Generate Graph', type: 'boolean', default: true, description: 'Generate graph on shutdown in backtest' },
    ],
  },
  {
    id: 'elliottwave',
    label: 'Elliott Wave',
    description: 'Elliott Wave oscillator with ATR stop-loss, Heikin-Ashi and trailing exits',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
      { key: 'stoploss', label: 'Stop Loss', type: 'number', default: 0.02, step: 0.001 },
      { key: 'MinInterval', label: 'Min Interval', type: 'select', default: '5m', options: ['1m', '5m', '15m', '30m'], description: 'Minimum interval for stop-loss and trailing exits' },
      { key: 'windowATR', label: 'ATR Window', type: 'number', default: 14, min: 1 },
      { key: 'windowQuick', label: 'Quick Window', type: 'number', default: 5, min: 1, description: 'Fast EWO window' },
      { key: 'windowSlow', label: 'Slow Window', type: 'number', default: 35, min: 1, description: 'Slow EWO window' },
      { key: 'useHeikinAshi', label: 'Heikin Ashi', type: 'boolean', default: false, description: 'Use Heikin-Ashi candles' },
      { key: 'drawGraph', label: 'Draw Graph', type: 'boolean', default: true },
    ],
  },
  {
    id: 'factorzoo',
    label: 'Factor Zoo',
    description: 'Multi-factor linear strategy with momentum and exit management',
    category: 'trend',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'Window', type: 'number', default: 20, min: 1, max: 500, required: true },
    ],
  },

  // ===================== Additional Mean Reversion Strategies =====================
  {
    id: 'ewo_dgtrd',
    label: 'EWO Divergence',
    description: 'Elliott Wave Oscillator divergence trading with CCI-Stochastic filters',
    category: 'mean-reversion',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
      { key: 'sigWin', label: 'Signal Window', type: 'number', default: 5, min: 1, max: 100, description: 'Signal window for EWO' },
      { key: 'stoploss', label: 'Stop Loss', type: 'number', default: 0.02, step: 0.001 },
      { key: 'useHeikinAshi', label: 'Heikin Ashi', type: 'boolean', default: false },
      { key: 'cciStochFilterHigh', label: 'CCI Filter High', type: 'number', default: 100, description: 'CCI Stochastic high filter' },
      { key: 'cciStochFilterLow', label: 'CCI Filter Low', type: 'number', default: -100, description: 'CCI Stochastic low filter' },
      { key: 'ewoChangeFilterHigh', label: 'EWO Filter High', type: 'number', default: 0, step: 0.01 },
      { key: 'ewoChangeFilterLow', label: 'EWO Filter Low', type: 'number', default: 0, step: 0.01 },
    ],
  },
  {
    id: 'harmonic',
    label: 'Harmonic',
    description: 'SHARK harmonic pattern detection with quantity-based entries',
    category: 'mean-reversion',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'Window', type: 'number', default: 20, min: 1, max: 500, required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'drawGraph', label: 'Draw Graph', type: 'boolean', default: true },
    ],
  },
  {
    id: 'irr',
    label: 'Negative Return Rate',
    description: 'Mean reversion using negative return rate indicator with quantity-based entries',
    category: 'mean-reversion',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
      { key: 'window', label: 'Window', type: 'number', default: 20, min: 1, max: 500, required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'drawGraph', label: 'Draw Graph', type: 'boolean', default: true },
    ],
  },

  // ===================== Additional DCA Strategies =====================
  {
    id: 'schedule',
    label: 'Scheduled Order',
    description: 'Submit orders on schedule with optional moving average conditions',
    category: 'dca',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['15m', '1h', '4h', '1d', '1w'] },
      { key: 'side', label: 'Side', type: 'select', default: 'buy', options: ['buy', 'sell'], required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'useLimitOrder', label: 'Limit Order', type: 'boolean', default: false, description: 'Use limit orders instead of market' },
      { key: 'minBaseBalance', label: 'Min Base Balance', type: 'number', default: 0, step: 0.0001, description: 'Minimum base balance to place sell orders' },
      { key: 'maxBaseBalance', label: 'Max Base Balance', type: 'number', default: 0, step: 0.0001, description: 'Maximum base balance for buy orders (0 = disabled)' },
    ],
  },

  // ===================== Additional Volatility Strategies =====================
  {
    id: 'xvs',
    label: 'Volume Surge',
    description: 'Trade on volume surge signals with EMA and pivot high confirmation',
    category: 'volatility',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'maxExposure', label: 'Max Exposure', type: 'number', default: 1, step: 0.01 },
      { key: 'volumeInterval', label: 'Volume Interval', type: 'select', default: '5m', options: ['1m', '5m', '15m', '1h'] },
      { key: 'volumeThreshold', label: 'Volume Threshold', type: 'number', default: 800, step: 1, description: 'Base asset volume threshold' },
      { key: 'stoploss', label: 'Stop Loss', type: 'number', default: 0.02, step: 0.001 },
    ],
  },

  // ===================== Additional Indicator Strategies =====================
  {
    id: 'techsignal',
    label: 'Tech Signal',
    description: 'Technical signal detection with support levels and funding rate monitoring',
    category: 'indicator',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
    ],
  },

  // ===================== Utility Strategies (Live-Only) =====================
  {
    id: 'autoborrow',
    label: 'Auto Borrow/Repay',
    description: 'Automatically manage margin borrowing and repayment based on margin level',
    category: 'utility',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget'],
    liveOnly: true,
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Check Interval', type: 'text', default: '5m', description: 'How often to check margin level' },
      { key: 'minMarginLevel', label: 'Min Margin Level', type: 'number', default: 1.5, step: 0.1, description: 'Repay when margin level drops below this' },
      { key: 'maxMarginLevel', label: 'Max Margin Level', type: 'number', default: 3.0, step: 0.1, description: 'Target margin level after repayment' },
      { key: 'autoRepayWhenDeposit', label: 'Auto Repay on Deposit', type: 'boolean', default: true },
    ],
  },
  {
    id: 'convert',
    label: 'Asset Converter',
    description: 'Automatically convert one asset to another (e.g., dust conversion)',
    category: 'utility',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    liveOnly: true,
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true, description: 'Trading pair for conversion' },
      { key: 'from', label: 'From Asset', type: 'text', default: 'BTC', required: true, description: 'Source asset to convert' },
      { key: 'to', label: 'To Asset', type: 'text', default: 'USDT', required: true, description: 'Target asset to receive' },
    ],
  },
  {
    id: 'deposit2transfer',
    label: 'Deposit Transfer',
    description: 'Automatically detect deposits and transfer assets to trading account',
    category: 'utility',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    liveOnly: true,
    fields: [
      { key: 'assets', label: 'Assets', type: 'text', default: 'USDT,BTC', required: true, description: 'Comma-separated asset list to watch' },
      { key: 'interval', label: 'Check Interval', type: 'text', default: '30s', description: 'How often to check for deposits' },
      { key: 'ignoreDust', label: 'Ignore Dust', type: 'boolean', default: true, description: 'Skip very small deposits' },
    ],
  },
  {
    id: 'support',
    label: 'Support Monitor',
    description: 'Detect support/resistance levels and trigger protective orders',
    category: 'utility',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },

  // ===================== Cross-Exchange Strategies =====================
  {
    id: 'xmaker',
    label: 'Cross-Exchange Market Maker',
    description: 'Market maker that hedges orders across two exchanges (maker + hedge sessions)',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'maker', label: 'Maker Session', futures: false },
      { name: 'hedge', label: 'Hedge Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001, description: 'Bid-ask spread ratio' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'updateInterval', label: 'Update Interval', type: 'text', default: '1m', description: 'Order book update interval' },
      { key: 'hedgeInterval', label: 'Hedge Interval', type: 'text', default: '5m', description: 'Hedge order check interval' },
    ],
  },
  {
    id: 'xbalance',
    label: 'Cross-Exchange Balance',
    description: 'Balance rebalancing across two exchange sessions',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'source', label: 'Source Session', futures: false },
      { name: 'target', label: 'Target Session', futures: false },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h', '1d'] },
    ],
  },
  {
    id: 'xalign',
    label: 'Cross-Exchange Align',
    description: 'Align positions across spot and futures sessions with arbitrage logic',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'spot', label: 'Spot Session', futures: false },
      { name: 'futures', label: 'Futures Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h'] },
    ],
  },
  {
    id: 'xpremium',
    label: 'Cross-Exchange Premium',
    description: 'Trade premium/discount between spot and futures across exchanges',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'spot', label: 'Spot Session', futures: false },
      { name: 'futures', label: 'Futures Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '5m', options: ['1m', '5m', '15m', '1h'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'xfixedmaker',
    label: 'Cross-Exchange Fixed Maker',
    description: 'Fixed spread market maker with cross-exchange hedging',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'maker', label: 'Maker Session', futures: false },
      { name: 'hedge', label: 'Hedge Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'halfSpread', label: 'Half Spread', type: 'number', default: 0.001, step: 0.0001, description: 'Half of the bid-ask spread' },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'xnav',
    label: 'Cross-Exchange NAV',
    description: 'Net asset value tracking and rebalancing across sessions',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'spot', label: 'Spot Session', futures: false },
      { name: 'futures', label: 'Futures Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['5m', '15m', '1h', '4h'] },
    ],
  },
  {
    id: 'xgap',
    label: 'Cross-Exchange Gap',
    description: 'Detect and trade price gaps across two exchanges',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'sessionA', label: 'Session A', futures: false },
      { name: 'sessionB', label: 'Session B', futures: false },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'xdepthmaker',
    label: 'Cross-Exchange Depth Maker',
    description: 'Depth-based market maker with cross-exchange order book comparison',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'maker', label: 'Maker Session', futures: false },
      { name: 'hedge', label: 'Hedge Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
    ],
  },
  {
    id: 'xfunding',
    label: 'Funding Rate Arbitrage',
    description: 'Capture funding rate differentials between spot and perpetual futures',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'spot', label: 'Spot Session', futures: false },
      { name: 'futures', label: 'Futures Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'xfundingv2',
    label: 'Funding Rate Arbitrage v2',
    description: 'Enhanced funding rate arbitrage with improved position management',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex', 'bybit', 'bitget', 'kucoin'],
    crossExchange: true,
    sessionRoles: [
      { name: 'spot', label: 'Spot Session', futures: false },
      { name: 'futures', label: 'Futures Session', futures: true },
    ],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
]

export function getStrategySchema(id: string): StrategySchema | undefined {
  return STRATEGY_SCHEMAS.find((s) => s.id === id)
}

export function getStrategyDefaults(id: string): Record<string, unknown> {
  const schema = getStrategySchema(id)
  if (!schema) return {}
  const defaults: Record<string, unknown> = {}
  for (const field of schema.fields) {
    defaults[field.key] = field.default
  }
  return defaults
}

export function ensureNumbers(schema: StrategySchema | undefined, config: Record<string, unknown>): Record<string, unknown> {
  if (!schema) return config
  const result = { ...config }
  for (const field of schema.fields) {
    if (field.type !== 'number') continue
    const v = result[field.key]
    if (v === '' || v === undefined || v === null) {
      delete result[field.key]
      continue
    }
    const num = Number(v)
    if (Number.isFinite(num)) result[field.key] = num
  }
  return result
}

export function getAllStrategies(): { id: string; label: string; description: string; category: string }[] {
  return STRATEGY_SCHEMAS.map((s) => ({ id: s.id, label: s.label, description: s.description, category: s.category }))
}

export function getStrategiesByCategory(opts?: { excludeLiveOnly?: boolean; excludeCrossExchange?: boolean }): Record<string, { id: string; label: string; description: string }[]> {
  const grouped: Record<string, { id: string; label: string; description: string }[]> = {}
  for (const s of STRATEGY_SCHEMAS) {
    if (opts?.excludeLiveOnly && s.liveOnly) continue
    if (opts?.excludeCrossExchange && s.category === 'cross-exchange') continue
    const cat = s.category
    if (!grouped[cat]) grouped[cat] = []
    grouped[cat]!.push({ id: s.id, label: s.label, description: s.description })
  }
  return grouped
}
