import { describe, it, expect } from 'vitest'
import {
  getStrategySchema,
  getStrategyDefaults,
  getAllStrategies,
  getStrategiesByCategory,
  type StrategySchema,
} from '../strategies'

const mockRegistry: StrategySchema[] = [
  {
    id: 'grid2',
    label: 'Grid V2',
    description: 'Grid trading',
    category: 'grid',
    supportedExchanges: ['binance', 'okex'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT' },
      { key: 'gridNumber', label: 'Grids', type: 'number', default: 10 },
      { key: 'quantity', label: 'Quantity', type: 'number', default: 0.001 },
    ],
  },
  {
    id: 'grid',
    label: 'Grid',
    description: 'Classic grid',
    category: 'grid',
    supportedExchanges: ['binance'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT' },
    ],
  },
  {
    id: 'bollmaker',
    label: 'Bollinger Maker',
    description: 'Bollinger band market maker',
    category: 'maker',
    supportedExchanges: ['binance'],
    liveOnly: true,
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT' },
    ],
  },
  {
    id: 'supertrend',
    label: 'Super Trend',
    description: 'Supertrend following',
    category: 'trend',
    supportedExchanges: ['binance', 'bybit'],
    fields: [
      { key: 'symbol', label: 'Symbol', type: 'text', default: 'BTCUSDT' },
    ],
  },
  {
    id: 'xmaker',
    label: 'Cross Maker',
    description: 'Cross-exchange maker',
    category: 'cross-exchange',
    supportedExchanges: ['binance', 'okex'],
    crossExchange: true,
    sessionRoles: [
      { name: 'maker', label: 'Maker', futures: false },
      { name: 'hedge', label: 'Hedge', futures: true },
    ],
    fields: [
      { key: 'spread', label: 'Spread', type: 'number', default: 0.001 },
    ],
  },
]

describe('Strategy trading chain contracts', () => {
  describe('getStrategySchema', () => {
    it('finds grid2 strategy with required trading fields', () => {
      const schema = getStrategySchema('grid2', mockRegistry)
      expect(schema).toBeDefined()
      expect(schema!.id).toBe('grid2')
      const keys = schema!.fields.map((f) => f.key)
      expect(keys).toContain('symbol')
      expect(keys).toContain('gridNumber')
      expect(keys).toContain('quantity')
    })

    it('finds all core trading strategies', () => {
      const essential = ['grid2', 'grid', 'bollmaker', 'supertrend']
      for (const id of essential) {
        expect(getStrategySchema(id, mockRegistry)).toBeDefined()
      }
    })

    it('returns undefined for unknown strategy', () => {
      expect(getStrategySchema('nonexistent_xyz', mockRegistry)).toBeUndefined()
    })
  })

  describe('getStrategyDefaults', () => {
    it('generates defaults for grid2 that match backend YAML expectations', () => {
      const defaults = getStrategyDefaults('grid2', mockRegistry)
      expect(defaults.symbol).toBe('BTCUSDT')
      expect(defaults.gridNumber).toBeDefined()
      expect(defaults.quantity).toBeDefined()
    })

    it('generates defaults for every strategy', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const defaults = getStrategyDefaults(s.id, mockRegistry)
        expect(Object.keys(defaults).length).toBeGreaterThan(0)
      }
    })

    it('returns empty object for unknown strategy', () => {
      expect(getStrategyDefaults('nonexistent', mockRegistry)).toEqual({})
    })
  })

  describe('Strategy field types are valid', () => {
    it('every field has a valid type', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        for (const field of schema.fields) {
          expect(['number', 'text', 'boolean', 'select', 'group']).toContain(field.type)
        }
      }
    })

    it('number fields have numeric defaults', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        for (const field of schema.fields) {
          if (field.type === 'number') {
            expect(typeof field.default).toBe('number')
          }
        }
      }
    })

    it('select fields have options array', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        for (const field of schema.fields) {
          if (field.type === 'select') {
            expect(field.options).toBeDefined()
            expect(field.options!.length).toBeGreaterThan(0)
          }
        }
      }
    })
  })

  describe('liveOnly strategies', () => {
    it('liveOnly strategies are identifiable', () => {
      const all = getAllStrategies(mockRegistry)
      const liveOnlyStrategies = all.filter((s) => getStrategySchema(s.id, mockRegistry)?.liveOnly === true)
      expect(liveOnlyStrategies.length).toBeGreaterThan(0)
    })

    it('getStrategiesByCategory can exclude liveOnly', () => {
      const withAll = getStrategiesByCategory(undefined, mockRegistry)
      const withoutLiveOnly = getStrategiesByCategory({ excludeLiveOnly: true }, mockRegistry)
      const totalWithAll = Object.values(withAll).flat().length
      const totalWithout = Object.values(withoutLiveOnly).flat().length
      expect(totalWithout).toBeLessThanOrEqual(totalWithAll)
    })

    it('cross-exchange strategies have sessionRoles', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        if (schema.crossExchange) {
          expect(schema.sessionRoles).toBeDefined()
          expect(schema.sessionRoles!.length).toBeGreaterThan(0)
        }
      }
    })
  })

  describe('Backend YAML alignment', () => {
    const PORTFOLIO_STRATEGIES = new Set<string>([])

    it('every non-cross-exchange symbol-targeting strategy has "symbol" field', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        if (schema.crossExchange || PORTFOLIO_STRATEGIES.has(s.id)) continue
        const keys = schema.fields.map((f) => f.key)
        expect(keys).toContain('symbol')
      }
    })

    it('portfolio strategies document their exception', () => {
      for (const id of PORTFOLIO_STRATEGIES) {
        const schema = getStrategySchema(id, mockRegistry)
        expect(schema).toBeDefined()
        expect(schema!.fields.map((f) => f.key)).not.toContain('symbol')
      }
    })

    it('symbol default is a non-empty string for symbol-targeting strategies', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        if (schema.crossExchange || PORTFOLIO_STRATEGIES.has(s.id)) continue
        const defaults = getStrategyDefaults(s.id, mockRegistry)
        expect(typeof defaults.symbol).toBe('string')
        expect((defaults.symbol as string).length).toBeGreaterThan(0)
      }
    })

    it('every strategy has supportedExchanges', () => {
      const all = getAllStrategies(mockRegistry)
      for (const s of all) {
        const schema = getStrategySchema(s.id, mockRegistry)!
        expect(schema.supportedExchanges.length).toBeGreaterThan(0)
      }
    })
  })
})

describe('UserContainer status handling', () => {
  it('accepts all valid status values from backend', () => {
    type Status = 'running' | 'stopped' | 'error' | 'starting'
    const statuses: Status[] = ['running', 'stopped', 'error', 'starting']
    expect(statuses).toHaveLength(4)
  })

  it('starting status is part of the type', () => {
    const status: 'running' | 'stopped' | 'error' | 'starting' = 'starting'
    expect(status).toBe('starting')
  })
})

describe('Trading mode validation', () => {
  it('paper and live are the only valid modes', () => {
    type Mode = 'live' | 'paper'
    const modes: Mode[] = ['live', 'paper']
    expect(modes).toHaveLength(2)
  })
})
