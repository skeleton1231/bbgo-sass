import { describe, it, expect } from 'vitest'
import { computePositionTags, computeFuturesPositionTags } from '../position-tags'

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

describe('computeFuturesPositionTags', () => {
  it('tags first BUY as openLong', () => {
    const trades = [{ side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' }]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openLong')
    expect(result[0]!.netPos).toBe(0.001)
    expect(result[0]!.direction).toBe('long')
  })

  it('tags first SELL as openShort', () => {
    const trades = [{ side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' }]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openShort')
    expect(result[0]!.netPos).toBe(-0.001)
    expect(result[0]!.direction).toBe('short')
  })

  it('tags closing BUY after short as closeShort', () => {
    const trades = [
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openShort')
    expect(result[1]!.tag).toBe('closeShort')
    expect(result[1]!.direction).toBe('flat')
  })

  it('tags closing SELL after long as closeLong', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openLong')
    expect(result[1]!.tag).toBe('closeLong')
  })

  it('tags addLong and reduceLong', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.002', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:15:59Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openLong')
    expect(result[1]!.tag).toBe('addLong')
    expect(result[2]!.tag).toBe('reduceLong')
    expect(result[2]!.netPos).toBe(0.002)
  })

  it('tags addShort and reduceShort', () => {
    const trades = [
      { side: 'SELL' as const, quantity: '0.002', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:15:59Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openShort')
    expect(result[1]!.tag).toBe('addShort')
    expect(result[2]!.tag).toBe('reduceShort')
    expect(result[2]!.netPos).toBe(-0.002)
  })

  it('handles flip from long to short', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.002', tradedAt: '2026-06-04T08:10:59Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openLong')
    expect(result[1]!.tag).toBe('flipLongToShort')
    expect(result[1]!.netPos).toBe(-0.001)
    expect(result[1]!.direction).toBe('short')
  })

  it('handles flip from short to long', () => {
    const trades = [
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'BUY' as const, quantity: '0.002', tradedAt: '2026-06-04T08:10:59Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    expect(result[0]!.tag).toBe('openShort')
    expect(result[1]!.tag).toBe('flipShortToLong')
    expect(result[1]!.netPos).toBe(0.001)
    expect(result[1]!.direction).toBe('long')
  })

  it('handles full long round trip', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
      { side: 'SELL' as const, quantity: '0.002', tradedAt: '2026-06-04T08:15:59Z' },
    ]
    const tags = computeFuturesPositionTags(trades).map((r) => r.tag)
    expect(tags).toEqual(['openLong', 'addLong', 'closeLong'])
  })

  it('handles full short round trip', () => {
    const trades = [
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:05:59Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:10:59Z' },
      { side: 'BUY' as const, quantity: '0.002', tradedAt: '2026-06-04T08:15:59Z' },
    ]
    const tags = computeFuturesPositionTags(trades).map((r) => r.tag)
    expect(tags).toEqual(['openShort', 'addShort', 'closeShort'])
  })

  it('returns empty for empty trades', () => {
    expect(computeFuturesPositionTags([])).toEqual([])
  })

  it('respects chronological ordering with DESC input', () => {
    const trades = [
      { side: 'BUY' as const, quantity: '0.001', tradedAt: '2026-06-04T12:00:00Z' },
      { side: 'SELL' as const, quantity: '0.001', tradedAt: '2026-06-04T08:00:00Z' },
    ]
    const result = computeFuturesPositionTags(trades)
    // SELL@08:00 is processed first (openShort), BUY@12:00 closes it
    expect(result[1]!.tag).toBe('openShort')
    expect(result[0]!.tag).toBe('closeShort')
  })
})
