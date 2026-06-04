import type { Time } from 'lightweight-charts'
import { FifoQueue } from './fifo-pnl'

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

const QUOTE_CURRENCIES = new Set(['USDT', 'USDC', 'BUSD', 'TUSD', 'DAI', 'FDUSD'])

function feeInQuote(fee: number, feeCurrency?: string, tradePrice?: number): number {
  if (!feeCurrency || QUOTE_CURRENCIES.has(feeCurrency)) return fee
  return tradePrice ? fee * tradePrice : fee
}

export function computePnlCurve(trades: PnlTrade[]): PnlPoint[] {
  if (trades.length === 0) return []

  const sorted = [...trades].sort(
    (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime()
  )

  const queue = new FifoQueue()
  let cumulative = 0
  const points: PnlPoint[] = []

  for (const trade of sorted) {
    const price = parseFloat(trade.price)
    const qty = parseFloat(trade.quantity)
    const fee = parseFloat(trade.fee || '0')

    if (trade.side === 'BUY') {
      queue.push(price, qty)
    } else {
      const { costBasis, remaining } = queue.match(qty)
      let matchedCost = costBasis
      if (remaining > 0) matchedCost += remaining * price
      cumulative += price * qty - matchedCost
    }

    cumulative -= feeInQuote(fee, trade.feeCurrency, price)

    const ts = Math.floor(new Date(trade.time).getTime() / 1000)
    const value = Math.round(cumulative * 100) / 100
    const last = points[points.length - 1]
    if (last && (last.time as number) === ts) {
      last.value = value
    } else {
      points.push({ time: ts as Time, value })
    }
  }

  return points
}
