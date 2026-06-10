export interface PositionTag {
  tag: 'open' | 'close' | 'add' | 'reduce' | 'trade' | null
  netPos: number
}

export interface FuturesPositionTag {
  tag: 'openLong' | 'closeLong' | 'openShort' | 'closeShort'
    | 'addLong' | 'reduceLong' | 'addShort' | 'reduceShort'
    | 'flipLongToShort' | 'flipShortToLong'
    | 'trade' | null
  netPos: number
  direction: 'long' | 'short' | 'flat'
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

/**
 * Compute futures-aware position action tags.
 *
 * Unlike spot mode where SELL always reduces a long position, futures allows
 * short selling — SELL can open a short, BUY can close a short. This function
 * tracks net position direction (positive=long, negative=short) and produces
 * direction-specific tags (openLong, closeShort, etc.).
 *
 * Position flips (long→short or short→long in a single trade) produce
 * `flipLongToShort` or `flipShortToLong` tags, preserving both the close
 * and the new open direction information.
 */
export function computeFuturesPositionTags(
  trades: Array<{ side: 'BUY' | 'SELL'; quantity: string; tradedAt?: string }>
): FuturesPositionTag[] {
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

  const tags: FuturesPositionTag[] = new Array(len)
  let net = 0

  for (const i of indices) {
    const { side, quantity } = trades[i]!
    const qty = side === 'BUY' ? parseFloat(quantity) : -parseFloat(quantity)
    const prev = net
    net += qty

    let tag: FuturesPositionTag['tag']
    const direction: FuturesPositionTag['direction'] =
      net > 0 ? 'long' : net < 0 ? 'short' : 'flat'

    if (prev === 0 && net > 0) {
      tag = 'openLong'
    } else if (prev === 0 && net < 0) {
      tag = 'openShort'
    } else if (prev > 0 && net === 0) {
      tag = 'closeLong'
    } else if (prev < 0 && net === 0) {
      tag = 'closeShort'
    } else if (prev > 0 && net < 0) {
      tag = 'flipLongToShort'
    } else if (prev < 0 && net > 0) {
      tag = 'flipShortToLong'
    } else if (prev > 0 && net > 0) {
      tag = qty > 0 ? 'addLong' : 'reduceLong'
    } else if (prev < 0 && net < 0) {
      tag = qty < 0 ? 'addShort' : 'reduceShort'
    } else {
      tag = null
    }

    tags[i] = { tag, netPos: net, direction }
  }

  return tags
}
