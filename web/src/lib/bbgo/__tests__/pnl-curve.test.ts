import { describe, it, expect } from 'vitest'
import { computePnlCurve, type PnlTrade } from '../pnl-curve'

const base: PnlTrade = {
  time: '2026-05-28T10:00:00Z',
  side: 'BUY',
  price: '72000',
  quantity: '0.001',
}

describe('computePnlCurve', () => {
  it('returns empty for no trades', () => {
    expect(computePnlCurve([])).toEqual([])
  })

  it('returns 0 for a single BUY (no realized P&L yet)', () => {
    const result = computePnlCurve([base])
    expect(result).toHaveLength(1)
    expect(result[0]!.value).toBe(0)
  })

  it('computes profit on BUY then SELL at higher price', () => {
    const trades: PnlTrade[] = [
      { ...base, side: 'BUY', price: '72000', quantity: '0.001', time: '2026-05-28T10:00:00Z' },
      { ...base, side: 'SELL', price: '73000', quantity: '0.001', time: '2026-05-28T11:00:00Z' },
    ]
    const result = computePnlCurve(trades)
    expect(result).toHaveLength(2)
    expect(result[0]!.value).toBe(0)
    expect(result[1]!.value).toBe(1)
  })

  it('computes loss on BUY then SELL at lower price', () => {
    const trades: PnlTrade[] = [
      { ...base, side: 'BUY', price: '72000', quantity: '0.001', time: '2026-05-28T10:00:00Z' },
      { ...base, side: 'SELL', price: '71000', quantity: '0.001', time: '2026-05-28T11:00:00Z' },
    ]
    const result = computePnlCurve(trades)
    expect(result[1]!.value).toBe(-1)
  })

  it('subtracts fees from cumulative P&L', () => {
    const trades: PnlTrade[] = [
      { ...base, side: 'BUY', price: '72000', quantity: '0.001', fee: '0.5', time: '2026-05-28T10:00:00Z' },
      { ...base, side: 'SELL', price: '73000', quantity: '0.001', fee: '0.5', time: '2026-05-28T11:00:00Z' },
    ]
    const result = computePnlCurve(trades)
    expect(result[0]!.value).toBe(-0.5) // only fee on BUY
    expect(result[1]!.value).toBe(0) // 1 - 0.5 (sell fee) - 0.5 (already deducted) = 0
  })

  it('handles partial SELL across multiple BUYs (FIFO)', () => {
    const trades: PnlTrade[] = [
      { ...base, side: 'BUY', price: '70000', quantity: '0.001', time: '2026-05-28T10:00:00Z' },
      { ...base, side: 'BUY', price: '72000', quantity: '0.001', time: '2026-05-28T10:30:00Z' },
      { ...base, side: 'SELL', price: '73000', quantity: '0.002', time: '2026-05-28T11:00:00Z' },
    ]
    const result = computePnlCurve(trades)
    expect(result[1]!.value).toBe(0)
    expect(result[2]!.value).toBe(4)
  })

  it('sorts trades by time before computing', () => {
    const trades: PnlTrade[] = [
      { ...base, side: 'SELL', price: '73000', quantity: '0.001', time: '2026-05-28T11:00:00Z' },
      { ...base, side: 'BUY', price: '72000', quantity: '0.001', time: '2026-05-28T10:00:00Z' },
    ]
    const result = computePnlCurve(trades)
    expect(result[0]!.value).toBe(0)
    expect(result[1]!.value).toBe(1)
  })

  it('handles multiple round-trip trades', () => {
    const trades: PnlTrade[] = [
      { ...base, side: 'BUY', price: '70000', quantity: '0.001', time: '2026-05-28T10:00:00Z' },
      { ...base, side: 'SELL', price: '71000', quantity: '0.001', time: '2026-05-28T11:00:00Z' },
      { ...base, side: 'BUY', price: '71500', quantity: '0.001', time: '2026-05-28T12:00:00Z' },
      { ...base, side: 'SELL', price: '72500', quantity: '0.001', time: '2026-05-28T13:00:00Z' },
    ]
    const result = computePnlCurve(trades)
    expect(result[1]!.value).toBe(1)
    expect(result[2]!.value).toBe(1)
    expect(result[3]!.value).toBe(2)
  })
})
