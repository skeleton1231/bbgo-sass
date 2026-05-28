export interface PositionTag {
  tag: 'open' | 'close' | 'add' | 'reduce' | 'trade' | null
  netPos: number
}

export function computePositionTags(
  trades: Array<{ side: 'BUY' | 'SELL'; quantity: string }>
): PositionTag[] {
  let net = 0
  return trades.map((t) => {
    const qty = t.side === 'BUY' ? parseFloat(t.quantity) : -parseFloat(t.quantity)
    const prev = net
    net += qty
    const tag = prev === 0 && net !== 0 ? 'open' as const
      : prev !== 0 && net === 0 ? 'close' as const
      : prev !== 0 ? (t.side === 'BUY' ? 'add' as const : 'reduce' as const)
      : null
    return { tag, netPos: net }
  })
}
