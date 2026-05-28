import type { Time } from 'lightweight-charts'

export interface PnlPoint {
  time: Time
  value: number
}

export interface PnlTrade {
  time: string
  side: 'BUY' | 'SELL'
  price: string
  quantity: string
  fee?: string
  feeCurrency?: string
}

/**
 * Compute cumulative realized P&L from trades using FIFO matching.
 * Each SELL is matched against previous BUYs to compute profit.
 * Returns an array of cumulative P&L points suitable for LineSeries overlay.
 */
export function computePnlCurve(trades: PnlTrade[]): PnlPoint[] {
  if (trades.length === 0) return []

  const sorted = [...trades].sort(
    (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime()
  )

  const longQueue: Array<{ price: number; qty: number }> = []
  let cumulative = 0
  const points: PnlPoint[] = []

  for (const trade of sorted) {
    const price = parseFloat(trade.price)
    const qty = parseFloat(trade.quantity)
    const fee = parseFloat(trade.fee || '0')

    if (trade.side === 'BUY') {
      longQueue.push({ price, qty })
    } else {
      let remaining = qty
      let matchedCost = 0
      while (remaining > 0 && longQueue.length > 0) {
        const head = longQueue[0]!
        const filled = Math.min(remaining, head.qty)
        matchedCost += filled * head.price
        head.qty -= filled
        remaining -= filled
        if (head.qty <= 1e-12) longQueue.shift()
      }
      cumulative += price * qty - matchedCost
    }

    cumulative -= fee

    const ts = new Date(trade.time).getTime() / 1000
    points.push({ time: ts as Time, value: Math.round(cumulative * 100) / 100 })
  }

  return points
}
