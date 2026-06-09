import { describe, it, expect } from 'vitest'
import { extractGridLines, extractStrategyStats, extractStrategyDetails, getStrategyCategory } from '../strategy-state'

describe('extractGridLines', () => {
  it('returns empty array when grid2 state is missing', () => {
    expect(extractGridLines({})).toEqual([])
  })

  it('returns empty array when grid2 has no upperPrice', () => {
    expect(extractGridLines({ grid2: { lowerPrice: 100, gridNumber: 5 } })).toEqual([])
  })

  it('returns empty array when grid2 has no gridNumber', () => {
    expect(extractGridLines({ grid2: { upperPrice: 200, lowerPrice: 100 } })).toEqual([])
  })

  it('generates correct number of grid lines', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    })
    expect(lines).toHaveLength(5)
  })

  it('generates evenly spaced price levels', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    })
    expect(lines).toHaveLength(5)
    expect(lines[0]!.price).toBe(100)
    expect(lines[1]!.price).toBe(125)
    expect(lines[2]!.price).toBe(150)
    expect(lines[3]!.price).toBe(175)
    expect(lines[4]!.price).toBe(200)
  })

  it('colors bounds darker than inner lines', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    })
    expect(lines[0]!.color).toContain('0.6')
    expect(lines[2]!.color).toContain('0.2')
    expect(lines[4]!.color).toContain('0.6')
  })

  it('highlights line near current price', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    }, 126)
    expect(lines[1]!.color).toContain('0.45')
  })

  it('formats labels with decimals for prices under 1000', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    })
    expect(lines[0]!.label).toBe('100.00')
  })

  it('formats labels without decimals for prices over 1000', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 50000, lowerPrice: 48000, gridNumber: 4 },
    })
    expect(lines[0]!.label).toBe('48000')
  })
})

describe('extractStrategyStats', () => {
  it('returns null when strategy field is missing', () => {
    expect(extractStrategyStats({})).toBeNull()
  })

  it('returns null when strategy sub-object is missing', () => {
    expect(extractStrategyStats({ strategy: 'grid2' })).toBeNull()
  })

  it('extracts stats from valid grid2 state', () => {
    const result = extractStrategyStats({
      strategy: 'grid2',
      grid2: {
        symbol: 'BTCUSDT',
        upperPrice: 50000,
        lowerPrice: 48000,
        gridNumber: 10,
        quantity: 0.001,
        Position: {
          base: '0.5',
          quote: '10000',
          averageCost: '49000',
          strategyInstanceID: 'instance-123',
        },
        GridProfitStats: {
          profit: '100',
          totalArbitrage: 5,
          startTime: '2026-01-01',
        },
      },
    })

    expect(result).toEqual({
      symbol: 'BTCUSDT',
      strategy: 'grid2',
      upperPrice: 50000,
      lowerPrice: 48000,
      gridNumber: 10,
      quantity: 0.001,
      base: 0.5,
      quote: 10000,
      averageCost: 49000,
      instanceId: 'instance-123',
      openedAt: '',
      stopLossPrice: 0,
      takeProfitPrice: 0,
    })
  })

  it('handles numeric Position fields', () => {
    const result = extractStrategyStats({
      strategy: 'grid2',
      grid2: {
        symbol: 'ETHUSDT',
        upperPrice: 3000,
        lowerPrice: 2000,
        gridNumber: 5,
        quantity: 0.1,
        Position: { base: 1.5, quote: 2000, strategyInstanceID: 'inst-456' },
      },
    })

    expect(result?.base).toBe(1.5)
    expect(result?.quote).toBe(2000)
  })

  it('handles missing optional fields with defaults', () => {
    const result = extractStrategyStats({
      strategy: 'grid2',
      grid2: {},
    })

    expect(result).toEqual({
      symbol: '',
      strategy: 'grid2',
      upperPrice: 0,
      lowerPrice: 0,
      gridNumber: 0,
      quantity: 0,
      base: 0,
      quote: 0,
      averageCost: 0,
      instanceId: '',
      openedAt: '',
      stopLossPrice: 0,
      takeProfitPrice: 0,
    })
  })
})

describe('getStrategyCategory', () => {
  it('classifies grid strategies', () => {
    expect(getStrategyCategory('grid2')).toBe('grid')
    expect(getStrategyCategory('grid')).toBe('grid')
    expect(getStrategyCategory('bollgrid')).toBe('grid')
    expect(getStrategyCategory('xhedgegrid')).toBe('grid')
  })

  it('classifies maker strategies', () => {
    expect(getStrategyCategory('bollmaker')).toBe('maker')
    expect(getStrategyCategory('fmaker')).toBe('maker')
    expect(getStrategyCategory('fixedmaker')).toBe('maker')
  })

  it('classifies dca strategies', () => {
    expect(getStrategyCategory('dca')).toBe('dca')
    expect(getStrategyCategory('dca2')).toBe('dca')
    expect(getStrategyCategory('dca3')).toBe('dca')
  })

  it('classifies trend strategies', () => {
    expect(getStrategyCategory('supertrend')).toBe('trend')
    expect(getStrategyCategory('emacross')).toBe('trend')
    expect(getStrategyCategory('pivotshort')).toBe('trend')
    expect(getStrategyCategory('swing')).toBe('trend')
  })

  it('defaults to other', () => {
    expect(getStrategyCategory('unknown')).toBe('other')
  })
})

describe('extractStrategyDetails', () => {
  it('returns null when strategy field is missing', () => {
    expect(extractStrategyDetails({})).toBeNull()
  })

  it('returns null when inner state is missing', () => {
    expect(extractStrategyDetails({ strategy: 'grid2' })).toBeNull()
  })

  it('extracts grid strategy details', () => {
    const result = extractStrategyDetails({
      strategy: 'grid2',
      grid2: {
        symbol: 'BTCUSDT',
        upperPrice: 50000,
        lowerPrice: 48000,
        gridNumber: 10,
        quantity: 0.001,
        stopLossPrice: 47000,
        takeProfitPrice: 51000,
      },
    })

    expect(result).not.toBeNull()
    expect(result!.strategy).toBe('grid2')
    expect(result!.category).toBe('grid')
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('range')
    expect(keys).toContain('gridNumber')
    expect(keys).toContain('quantity')
    expect(keys).toContain('stopLoss')
    expect(keys).toContain('takeProfit')
  })

  it('extracts trend strategy details for pivotshort', () => {
    const result = extractStrategyDetails({
      strategy: 'pivotshort',
      pivotshort: {
        symbol: 'BTCUSDT',
        interval: '1h',
        quantity: 0.001,
        leverage: 5,
        Position: { base: -0.002, quote: -130, averageCost: 65000 },
      },
    })

    expect(result).not.toBeNull()
    expect(result!.strategy).toBe('pivotshort')
    expect(result!.category).toBe('trend')
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('interval')
    expect(keys).toContain('quantity')
    expect(keys).toContain('leverage')
    expect(keys).toContain('pos.base')
    expect(keys).toContain('pos.avgCost')
    expect(keys).not.toContain('range')
    expect(keys).not.toContain('gridNumber')
  })

  it('extracts maker strategy details', () => {
    const result = extractStrategyDetails({
      strategy: 'bollmaker',
      bollmaker: {
        symbol: 'ETHUSDT',
        interval: '15m',
        spread: 10,
        quantity: 0.5,
      },
    })

    expect(result).not.toBeNull()
    expect(result!.category).toBe('maker')
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('spread')
    expect(keys).toContain('quantity')
    expect(keys).toContain('interval')
  })

  it('extracts dca strategy details', () => {
    const result = extractStrategyDetails({
      strategy: 'dca',
      dca: {
        symbol: 'BTCUSDT',
        interval: '1d',
        budget: 1000,
        quantity: 0.001,
      },
    })

    expect(result).not.toBeNull()
    expect(result!.category).toBe('dca')
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('budget')
    expect(keys).toContain('interval')
    expect(keys).toContain('quantity')
  })

  it('extracts emacross with fast/slow length', () => {
    const result = extractStrategyDetails({
      strategy: 'emacross',
      emacross: {
        symbol: 'BTCUSDT',
        fastLength: 7,
        slowLength: 25,
        quantity: 0.001,
      },
    })

    expect(result).not.toBeNull()
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('fastLength')
    expect(keys).toContain('slowLength')
  })

  it('falls back to default for unknown strategies', () => {
    const result = extractStrategyDetails({
      strategy: 'custom',
      custom: {
        symbol: 'BTCUSDT',
        someField: 42,
        Position: { base: 0, quote: 0, averageCost: 0 },
      },
    })

    expect(result).not.toBeNull()
    expect(result!.category).toBe('other')
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('someField')
  })

  it('includes position fields when present', () => {
    const result = extractStrategyDetails({
      strategy: 'supertrend',
      supertrend: {
        symbol: 'BTCUSDT',
        interval: '1h',
        Position: { base: 0.1, quote: 5000, averageCost: 50000 },
      },
    })

    expect(result).not.toBeNull()
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('pos.base')
    expect(keys).toContain('pos.avgCost')
    expect(keys).toContain('pos.quote')
  })

  it('includes grid profit stats when present', () => {
    const result = extractStrategyDetails({
      strategy: 'grid2',
      grid2: {
        symbol: 'BTCUSDT',
        upperPrice: 50000,
        lowerPrice: 48000,
        gridNumber: 10,
        quantity: 0.001,
        GridProfitStats: { profit: 50, totalArbitrage: 10 },
      },
    })

    expect(result).not.toBeNull()
    const keys = result!.fields.map((f) => f.key)
    expect(keys).toContain('gridProfit')
    expect(keys).toContain('totalArbitrage')
  })
})
