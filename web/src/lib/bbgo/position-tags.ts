export interface PositionTag {
  tag: 'open' | 'close' | 'add' | 'reduce' | 'trade' | null
  netPos: number
}

/**
 * Compute position action tags for a list of trades.
 *
 * Trades are processed chronologically (tradedAt ASC). When trades share the
 * same timestamp (common in grid strategies where a kline close fills multiple
 * orders), higher array indices are processed first — this matches the bbgo API
 * which returns trades in DESC order (newest first), so reversing within
 * same-timestamp groups recreates the original gid ASC insertion order.
 */
export function computePositionTags(
  trades: Array<{ side: 'BUY' | 'SELL'; quantity: string; tradedAt?: string }>
): PositionTag[] {
  if (trades.length === 0) return []

  const len = trades.length
  const indices = Array.from({ length: len }, (_, i) => i)
  indices.sort((a, b) => {
    const ta = trades[a]!.tradedAt ?? ''
    const tb = trades[b]!.tradedAt ?? ''
    if (ta < tb) return -1
    if (ta > tb) return 1
    return b - a
  })

  const tags: PositionTag[] = new Array(len)
  let net = 0

  for (const i of indices) {
    const { side, quantity } = trades[i]!
    const qty = side === 'BUY' ? parseFloat(quantity) : -parseFloat(quantity)
    const prev = net
    net += qty
    const tag = prev === 0 && net !== 0 ? 'open' as const
      : prev !== 0 && net === 0 ? 'close' as const
      : prev !== 0 ? (side === 'BUY' ? 'add' as const : 'reduce' as const)
      : null
    tags[i] = { tag, netPos: net }
  }

  return tags
}
