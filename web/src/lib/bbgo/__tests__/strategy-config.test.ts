import { describe, it, expect } from 'vitest'
import { nestConfig, ensureTypes, getStrategyDefaults } from '../strategies'
import type { StrategySchema } from '../strategies'

describe('nestConfig', () => {
  it('converts flat dotted keys to nested objects', () => {
    const result = nestConfig({
      'defaultBollinger.bandWidth': 2,
      'defaultBollinger.interval': '1m',
      symbol: 'BTCUSDT',
    })
    expect(result).toEqual({
      defaultBollinger: { bandWidth: 2, interval: '1m' },
      symbol: 'BTCUSDT',
    })
  })

  it('handles deeply nested keys (3 levels)', () => {
    const result = nestConfig({
      'a.b.c': 42,
    })
    expect(result).toEqual({ a: { b: { c: 42 } } })
  })

  it('merges sibling keys under the same parent', () => {
    const result = nestConfig({
      'config.leverage': 10,
      'config.marginType': 'isolated',
      'config.symbol': 'BTCUSDT',
    })
    expect(result).toEqual({
      config: { leverage: 10, marginType: 'isolated', symbol: 'BTCUSDT' },
    })
  })

  it('skips undefined, null, and empty string values', () => {
    const result = nestConfig({
      symbol: 'BTCUSDT',
      interval: '',
      quantity: null,
      leverage: undefined,
    })
    expect(result).toEqual({ symbol: 'BTCUSDT' })
  })

  it('preserves numeric zero as a valid value', () => {
    const result = nestConfig({ gridNumber: 0 })
    expect(result).toEqual({ gridNumber: 0 })
  })

  it('preserves boolean false as a valid value', () => {
    const result = nestConfig({ enabled: false })
    expect(result).toEqual({ enabled: false })
  })

  it('handles empty config', () => {
    expect(nestConfig({})).toEqual({})
  })

  it('produces independent nested objects (no shared references)', () => {
    const result = nestConfig({
      'session1.leverage': 5,
      'session2.leverage': 10,
    })
    expect(result).toEqual({
      session1: { leverage: 5 },
      session2: { leverage: 10 },
    })
  })
})

describe('ensureTypes', () => {
  const schema: StrategySchema = {
    id: 'test',
    label: 'Test',
    description: '',
    category: 'grid',
    supportedExchanges: ['binance'],
    fields: [
      { key: 'quantity', label: 'Qty', type: 'number', default: 0.001 },
      { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT' },
      { key: 'interval', label: 'Interval', type: 'select', default: '1m', options: ['1m', '5m'] },
    ],
  }

  it('converts string number to number type', () => {
    const result = ensureTypes(schema, { quantity: '0.5' })
    expect(result.quantity).toBe(0.5)
    expect(typeof result.quantity).toBe('number')
  })

  it('converts string "true" to boolean true', () => {
    const result = ensureTypes(schema, { enabled: 'true' })
    expect(result.enabled).toBe(true)
  })

  it('converts string "false" to boolean false', () => {
    const result = ensureTypes(schema, { enabled: 'false' })
    expect(result.enabled).toBe(false)
  })

  it('preserves numeric values that are already numbers', () => {
    const result = ensureTypes(schema, { quantity: 0.5 })
    expect(result.quantity).toBe(0.5)
  })

  it('preserves boolean values that are already booleans', () => {
    const result = ensureTypes(schema, { enabled: false })
    expect(result.enabled).toBe(false)
  })

  it('deletes empty string number fields', () => {
    const result = ensureTypes(schema, { quantity: '' })
    expect(result).not.toHaveProperty('quantity')
  })

  it('deletes undefined number fields', () => {
    const result = ensureTypes(schema, { quantity: undefined })
    expect(result).not.toHaveProperty('quantity')
  })

  it('deletes null number fields', () => {
    const result = ensureTypes(schema, { quantity: null })
    expect(result).not.toHaveProperty('quantity')
  })

  it('leaves text fields untouched', () => {
    const result = ensureTypes(schema, { symbol: 'ETHUSDT' })
    expect(result.symbol).toBe('ETHUSDT')
  })

  it('leaves select fields untouched', () => {
    const result = ensureTypes(schema, { interval: '5m' })
    expect(result.interval).toBe('5m')
  })

  it('coerces non-boolean, non-string truthy values to boolean', () => {
    const result = ensureTypes(schema, { enabled: 1 })
    expect(result.enabled).toBe(true)
  })

  it('coerces non-boolean, non-string falsy values to boolean', () => {
    const result = ensureTypes(schema, { enabled: 0 })
    expect(result.enabled).toBe(false)
  })

  it('returns config unchanged when schema is undefined', () => {
    const config = { quantity: '0.5', symbol: 'BTC' }
    expect(ensureTypes(undefined, config)).toEqual(config)
  })

  it('handles NaN gracefully (keeps original if not finite)', () => {
    const result = ensureTypes(schema, { quantity: 'abc' })
    expect(result.quantity).toBe('abc')
  })
})

describe('getStrategyDefaults', () => {
  const schema: StrategySchema = {
    id: 'test',
    label: 'Test',
    description: '',
    category: 'grid',
    supportedExchanges: [],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT' },
      { key: 'quantity', label: 'Qty', type: 'number', default: 0.001 },
      { key: 'defaultBollinger.bandWidth', label: 'BW', type: 'number', default: 2 },
      { key: 'defaultBollinger.interval', label: 'IV', type: 'text', default: '1m' },
    ],
  }

  it('builds flat defaults from schema fields', () => {
    const defaults = getStrategyDefaults('test', [schema])
    expect(defaults.symbol).toBe('BTCUSDT')
    expect(defaults.quantity).toBe(0.001)
  })

  it('builds nested defaults from dotted keys', () => {
    const defaults = getStrategyDefaults('test', [schema])
    expect(defaults).toHaveProperty('defaultBollinger')
    expect((defaults.defaultBollinger as Record<string, unknown>).bandWidth).toBe(2)
    expect((defaults.defaultBollinger as Record<string, unknown>).interval).toBe('1m')
  })

  it('returns empty object when schema not found', () => {
    expect(getStrategyDefaults('nonexistent', [schema])).toEqual({})
  })

  it('returns empty object when registry is empty', () => {
    expect(getStrategyDefaults('test', [])).toEqual({})
  })

  it('returns empty object when registry is undefined', () => {
    expect(getStrategyDefaults('test')).toEqual({})
  })
})
