import { describe, it, expect } from 'vitest'

function fmt(v: number, digits = 2): string {
  return v.toLocaleString('en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

function fmtVol(v: number): string {
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(2)}M`
  if (v >= 1_000) return `${(v / 1_000).toFixed(2)}K`
  return v.toFixed(2)
}

function calcChange(close: number, ref: number) {
  const change = close - ref
  const pct = ref > 0 ? (change / ref) * 100 : 0
  return { change, pct, isUp: change >= 0 }
}

describe('fmt (price formatter)', () => {
  it('formats with default 2 decimal places', () => {
    expect(fmt(123.456)).toBe('123.46')
  })

  it('formats with custom decimal places', () => {
    expect(fmt(123.4567, 4)).toBe('123.4567')
  })

  it('adds thousand separators', () => {
    expect(fmt(73185.52)).toBe('73,185.52')
  })

  it('pads zeros', () => {
    expect(fmt(100)).toBe('100.00')
  })
})

describe('fmtVol (volume formatter)', () => {
  it('formats millions', () => {
    expect(fmtVol(1_500_000)).toBe('1.50M')
  })

  it('formats thousands', () => {
    expect(fmtVol(5_000)).toBe('5.00K')
  })

  it('formats small volumes', () => {
    expect(fmtVol(500)).toBe('500.00')
  })

  it('formats exactly 1M', () => {
    expect(fmtVol(1_000_000)).toBe('1.00M')
  })

  it('formats exactly 1K', () => {
    expect(fmtVol(1_000)).toBe('1.00K')
  })
})

describe('calcChange (price change calculator)', () => {
  it('calculates positive change', () => {
    const result = calcChange(105, 100)
    expect(result.change).toBe(5)
    expect(result.pct).toBeCloseTo(5)
    expect(result.isUp).toBe(true)
  })

  it('calculates negative change', () => {
    const result = calcChange(95, 100)
    expect(result.change).toBe(-5)
    expect(result.pct).toBeCloseTo(-5)
    expect(result.isUp).toBe(false)
  })

  it('handles zero ref price', () => {
    const result = calcChange(100, 0)
    expect(result.pct).toBe(0)
    expect(result.isUp).toBe(true)
  })

  it('handles no change', () => {
    const result = calcChange(100, 100)
    expect(result.change).toBe(0)
    expect(result.pct).toBe(0)
    expect(result.isUp).toBe(true)
  })
})
