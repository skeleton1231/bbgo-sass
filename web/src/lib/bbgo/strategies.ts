export type FieldType = 'number' | 'text' | 'boolean' | 'select'

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

export interface StrategySchema {
  id: string
  label: string
  fields: FieldDef[]
}

const STRATEGY_SCHEMAS: StrategySchema[] = [
  {
    id: 'grid2',
    label: 'Grid Trading',
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true, description: 'Trading pair' },
      { key: 'gridNumber', label: 'Grid Number', type: 'number', default: 10, min: 2, max: 100, required: true },
      { key: 'upperPrice', label: 'Upper Price', type: 'number', default: 70000, step: 0.01, required: true },
      { key: 'lowerPrice', label: 'Lower Price', type: 'number', default: 50000, step: 0.01, required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'profitSpread', label: 'Profit Spread', type: 'number', default: 0.005, step: 0.001 },
    ],
  },
  {
    id: 'bollmaker',
    label: 'Bollinger Maker',
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'bidQuantity', label: 'Bid Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'askQuantity', label: 'Ask Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
      { key: 'minProfitSpread', label: 'Min Profit Spread', type: 'number', default: 0.001, step: 0.0001 },
    ],
  },
  {
    id: 'supertrend',
    label: 'Supertrend',
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'supertrendMultiplier', label: 'Multiplier', type: 'number', default: 3, step: 0.1, min: 0.1 },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'leverage', label: 'Leverage', type: 'number', default: 1, min: 1, max: 125 },
    ],
  },
  {
    id: 'emacross',
    label: 'EMA Cross',
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'fastWindow', label: 'Fast EMA Period', type: 'number', default: 9, min: 1, max: 200 },
      { key: 'slowWindow', label: 'Slow EMA Period', type: 'number', default: 21, min: 1, max: 200 },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
    ],
  },
  {
    id: 'xmaker',
    label: 'Cross Exchange Maker',
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001, step: 0.0001, required: true },
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001, step: 0.0001 },
    ],
  },
  {
    id: 'dca',
    label: 'Dollar Cost Average',
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT', required: true },
      { key: 'interval', label: 'Interval', type: 'select', default: '1h', options: ['1m', '5m', '15m', '1h', '4h', '1d'] },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 100, step: 0.01, required: true },
      { key: 'maxOrders', label: 'Max Orders', type: 'number', default: 5, min: 1, max: 50 },
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

export function getAllStrategies(): { id: string; label: string }[] {
  return STRATEGY_SCHEMAS.map((s) => ({ id: s.id, label: s.label }))
}
