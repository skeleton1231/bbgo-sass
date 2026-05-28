import { describe, it, expect } from 'vitest'
import { computePositionTags } from '../position-tags'

describe('computePositionTags', () => {
  it('tags first BUY as open', () => {
    const trades = [{ side: 'BUY' as const, quantity: '0.001' }]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[0]!.netPos).toBe(0.001)
  })

  it('tags closing SELL as close', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001' },
      { side: 'SELL' as const, quantity: '0.001' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('close')
  })

  it('tags add position correctly', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001' },
      { side: 'BUY' as const, quantity: '0.001' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('add')
    expect(result[1]!.netPos).toBe(0.002)
  })

  it('tags reduce position correctly', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.002' },
      { side: 'SELL' as const, quantity: '0.001' },
      { side: 'SELL' as const, quantity: '0.001' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('reduce')
    expect(result[2]!.tag).toBe('close')
  })

  it('handles multiple round trips', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001' },
      { side: 'SELL' as const, quantity: '0.001' },
      { side: 'SELL' as const, quantity: '0.001' },
      { side: 'BUY' as const, quantity: '0.001' },
    ]
    const tags = computePositionTags(trades).map((r) => r.tag)
    expect(tags).toEqual(['open', 'close', 'open', 'close'])
  })

  it('returns empty for empty trades', () => {
    expect(computePositionTags([])).toEqual([])
  })
})
