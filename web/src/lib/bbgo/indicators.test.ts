import { describe, it, expect } from 'vitest'
import { computeSMA, computeEMA, computeBollingerBands } from './indicators'
import type { Time } from 'lightweight-charts'

const closes: Array<{ time: Time; close: number }> = [
  { time: 1 as Time, close: 10 },
  { time: 2 as Time, close: 20 },
  { time: 3 as Time, close: 30 },
  { time: 4 as Time, close: 40 },
  { time: 5 as Time, close: 50 },
  { time: 6 as Time, close: 60 },
  { time: 7 as Time, close: 70 },
  { time: 8 as Time, close: 80 },
  { time: 9 as Time, close: 90 },
  { time: 10 as Time, close: 100 },
]

describe('computeSMA', () => {
  it('returns empty when insufficient data', () => {
    expect(computeSMA(closes.slice(0, 2), 3)).toEqual([])
  })

  it('computes SMA(3) correctly', () => {
    const result = computeSMA(closes, 3)
    expect(result).toHaveLength(8)
    expect(result[0]).toEqual({ time: 3, value: 20 })
    expect(result[1]).toEqual({ time: 4, value: 30 })
    expect(result[7]).toEqual({ time: 10, value: 90 })
  })

  it('computes SMA(5) correctly', () => {
    const result = computeSMA(closes, 5)
    expect(result).toHaveLength(6)
    expect(result[0]).toEqual({ time: 5, value: 30 })
    expect(result[5]).toEqual({ time: 10, value: 80 })
  })

  it('handles single period', () => {
    const result = computeSMA(closes, 1)
    expect(result).toHaveLength(10)
    expect(result[0]).toEqual({ time: 1, value: 10 })
  })
})

describe('computeEMA', () => {
  it('returns empty when insufficient data', () => {
    expect(computeEMA(closes.slice(0, 2), 3)).toEqual([])
  })

  it('first EMA value equals SMA seed', () => {
    const result = computeEMA(closes, 3)
    expect(result[0]!).toEqual({ time: 3, value: 20 })
  })

  it('applies exponential weighting', () => {
    const result = computeEMA(closes, 3)
    const k = 2 / (3 + 1)
    expect(result[1]!.value).toBeCloseTo(40 * k + 20 * (1 - k), 10)
    expect(result[1]!.value).toBe(30)
  })

  it('produces correct number of points', () => {
    const result = computeEMA(closes, 5)
    expect(result).toHaveLength(6)
    expect(result[0]!.time).toBe(5)
  })
})

describe('computeBollingerBands', () => {
  it('returns empty when insufficient data', () => {
    const result = computeBollingerBands(closes.slice(0, 2), 5, 2)
    expect(result.upper).toEqual([])
    expect(result.middle).toEqual([])
    expect(result.lower).toEqual([])
  })

  it('middle band equals SMA', () => {
    const { middle } = computeBollingerBands(closes, 3, 2)
    const sma = computeSMA(closes, 3)
    expect(middle).toEqual(sma)
  })

  it('upper band is above middle, lower band below', () => {
    const { upper, middle, lower } = computeBollingerBands(closes, 5, 2)
    expect(upper.length).toBe(middle.length)
    expect(lower.length).toBe(middle.length)
    for (let i = 0; i < middle.length; i++) {
      expect(upper[i]!.value).toBeGreaterThan(middle[i]!.value)
      expect(lower[i]!.value).toBeLessThan(middle[i]!.value)
    }
  })

  it('bands widen with volatility', () => {
    const volatile: Array<{ time: Time; close: number }> = [
      { time: 1 as Time, close: 100 },
      { time: 2 as Time, close: 50 },
      { time: 3 as Time, close: 150 },
      { time: 4 as Time, close: 50 },
      { time: 5 as Time, close: 150 },
    ]
    const { upper, lower } = computeBollingerBands(volatile, 5, 2)
    const bandwidth = upper[0]!.value - lower[0]!.value
    expect(bandwidth).toBeGreaterThan(0)
  })
})
