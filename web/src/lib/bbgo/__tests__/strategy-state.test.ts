import { describe, it, expect } from 'vitest'
import { extractGridLines, extractStrategyStats } from '../strategy-state'

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
    expect(lines[0].price).toBe(100)
    expect(lines[1].price).toBe(125)
    expect(lines[2].price).toBe(150)
    expect(lines[3].price).toBe(175)
    expect(lines[4].price).toBe(200)
  })

  it('colors bounds darker than inner lines', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    })
    expect(lines[0].color).toContain('0.6')
    expect(lines[2].color).toContain('0.2')
    expect(lines[4].color).toContain('0.6')
  })

  it('highlights line near current price', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    }, 126)
    expect(lines[1].color).toContain('0.45')
  })

  it('formats labels with decimals for prices under 1000', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 200, lowerPrice: 100, gridNumber: 4 },
    })
    expect(lines[0].label).toBe('100.00')
  })

  it('formats labels without decimals for prices over 1000', () => {
    const lines = extractGridLines({
      grid2: { upperPrice: 50000, lowerPrice: 48000, gridNumber: 4 },
    })
    expect(lines[0].label).toBe('48000')
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
      instanceId: 'instance-123',
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
      instanceId: '',
    })
  })
})
