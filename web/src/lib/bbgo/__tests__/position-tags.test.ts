import { describe, it, expect } from 'vitest'
import { computePositionTags } from '../position-tags'

describe('computePositionTags', () => {
  it('tags first BUY as open', () => {
    const trades = [{ side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' }]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[0]!.netPos).toBe(0.001)
  })

  it('tags closing SELL as close', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('close')
  })

  it('tags add position correctly', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('add')
    expect(result[1]!.netPos).toBe(0.002)
  })

  it('tags reduce position correctly', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.002', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:15:59Z' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('reduce')
    expect(result[2]!.tag).toBe('close')
  })

  it('handles multiple round trips', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:15:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:20:59Z' },
    ]
    const tags = computePositionTags(trades).map((r) => r.tag)
    expect(tags).toEqual(['open', 'close', 'open', 'close'])
  })

  it('returns empty for empty trades', () => {
    expect(computePositionTags([])).toEqual([])
  })

  it('computes correct tags when trades are in DESC order (newest first)', () => {
    // ASC order: BUY@08:05 → SELL@09:34 → SELL@09:45 → BUY@12:54 → BUY@12:55
    // Sequence: open → close → open(short) → close → open
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T12:55:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T12:54:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T09:45:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T09:34:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('open')
    expect(result[1]!.tag).toBe('close')
    expect(result[2]!.tag).toBe('open')
    expect(result[3]!.tag).toBe('close')
    expect(result[4]!.tag).toBe('open')
  })

  it('computes correct tags when trades are out of order', () => {
    const trades = [
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T09:00:00Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:00:00Z' },
    ]
    const result = computePositionTags(trades)
    expect(result[0]!.tag).toBe('close')
    expect(result[1]!.tag).toBe('open')
  })

  it('orders same-timestamp trades by reversed index (matching SQLite gid ASC)', () => {
    // API returns DESC order (index 0 = newest). Trades share same tradedAt but have different gids.
    // SQLite processes them gid ASC. Within same-timestamp group, reversing the DESC array order
    // (b - a tie-break) recreates the gid ASC sequence.
    const trades = [
      { side: 'BUY' as const, quantity: '0.003', tradedAt: '2026-06-04T12:00:59.999Z' }, // gid highest
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T12:00:59.999Z' },
      { side: 'BUY' as const, quantity: '0.002', tradedAt: '2026-06-04T12:00:59.999Z' }, // gid lowest
    ]
    const result = computePositionTags(trades)
    // Processed order (gid ASC): index 2 (BUY 0.002) → index 1 (SELL 0.001) → index 0 (BUY 0.003)
    expect(result[2]!.tag).toBe('open')
    expect(result[2]!.netPos).toBe(0.002)
    expect(result[1]!.tag).toBe('reduce')
    expect(result[1]!.netPos).toBe(0.001)
    expect(result[0]!.tag).toBe('add')
    expect(result[0]!.netPos).toBe(0.004)
  })

  it('handles trades without tradedAt field (reverses array order on tie)', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001' },
      { side: 'BUY' as const, quantity: '0.001' },
    ]
    const result = computePositionTags(trades)
    // Without tradedAt, both tie → b-a reverses order: index 1 first (open), index 0 second (add)
    expect(result[1]!.tag).toBe('open')
    expect(result[0]!.tag).toBe('add')
  })
})
