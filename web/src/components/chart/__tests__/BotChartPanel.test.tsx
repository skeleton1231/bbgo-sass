import { describe, it, expect } from 'vitest'

function computeAllIndicators(
  indicatorLines: Array<{ id: string }>,
  pnlLine: { id: string } | null,
  showPnlCurve: boolean
) {
  return pnlLine && showPnlCurve ? [...indicatorLines, pnlLine] : indicatorLines
}

function computePositionTag(
  trades: Array<{ side: 'BUY' | 'SELL'; quantity: string }>
): Array<{ tag: 'open' | 'close' | null; netPos: number }> {
  let net = 0
  return trades.map((t) => {
    const qty = t.side === 'BUY' ? parseFloat(t.quantity) : -parseFloat(t.quantity)
    const prev = net
    net += qty
    const tag = prev === 0 && net !== 0 ? 'open' as const
      : prev !== 0 && net === 0 ? 'close' as const
      : null
    return { tag, netPos: net }
  })
}

describe('BotChartPanel logic', () => {
  describe('allIndicators', () => {
    const lines = [{ id: 'sma-20' }, { id: 'ema-50' }]

    it('includes pnlLine when showPnlCurve is true', () => {
      const pnl = { id: 'pnl-curve' }
      const result = computeAllIndicators(lines, pnl, true)
      expect(result).toHaveLength(3)
      expect(result[2]!.id).toBe('pnl-curve')
    })

    it('excludes pnlLine when showPnlCurve is false', () => {
      const pnl = { id: 'pnl-curve' }
      const result = computeAllIndicators(lines, pnl, false)
      expect(result).toHaveLength(2)
    })

    it('returns only indicatorLines when pnlLine is null', () => {
      const result = computeAllIndicators(lines, null, true)
      expect(result).toHaveLength(2)
    })

    it('returns empty array for no indicators and no pnl', () => {
      const result = computeAllIndicators([], null, true)
      expect(result).toEqual([])
    })
  })

  describe('position tags', () => {
    it('tags first BUY as open', () => {
      const trades = [{ side: 'BUY' as const, quantity: '0.001' }]
      const result = computePositionTag(trades)
      expect(result[0]!.tag).toBe('open')
      expect(result[0]!.netPos).toBe(0.001)
    })

    it('tags closing SELL as close', () => {
      const trades = [
        { side: 'BUY' as const, quantity: '0.001' },
        { side: 'SELL' as const, quantity: '0.001' },
      ]
      const result = computePositionTag(trades)
      expect(result[0]!.tag).toBe('open')
      expect(result[1]!.tag).toBe('close')
      expect(result[1]!.netPos).toBe(0)
    })

    it('tags add/reduce as null', () => {
      const trades = [
        { side: 'BUY' as const, quantity: '0.001' },
        { side: 'BUY' as const, quantity: '0.001' },
      ]
      const result = computePositionTag(trades)
      expect(result[0]!.tag).toBe('open')
      expect(result[1]!.tag).toBeNull()
      expect(result[1]!.netPos).toBe(0.002)
    })

    it('handles multiple round trips', () => {
      const trades = [
        { side: 'BUY' as const, quantity: '0.001' },
        { side: 'SELL' as const, quantity: '0.001' },
        { side: 'SELL' as const, quantity: '0.001' },
        { side: 'BUY' as const, quantity: '0.001' },
      ]
      const tags = computePositionTag(trades).map((r) => r.tag)
      expect(tags).toEqual(['open', 'close', 'open', 'close'])
    })

    it('tracks net position correctly for partial close', () => {
      const trades = [
        { side: 'BUY' as const, quantity: '0.002' },
        { side: 'SELL' as const, quantity: '0.001' },
        { side: 'SELL' as const, quantity: '0.001' },
      ]
      const result = computePositionTag(trades)
      expect(result[0]!.netPos).toBe(0.002)
      expect(result[1]!.netPos).toBe(0.001)
      expect(result[1]!.tag).toBeNull()
      expect(result[2]!.netPos).toBe(0)
      expect(result[2]!.tag).toBe('close')
    })
  })
})
