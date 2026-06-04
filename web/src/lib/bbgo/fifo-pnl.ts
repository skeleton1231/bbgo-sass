import type { BBGoTrade } from './queries'

export class FifoQueue {
  private items: Array<{ price: number; quantity: number }> = []
  private head = 0

  push(price: number, quantity: number): void {
    this.items.push({ price, quantity })
  }

  match(quantity: number): { costBasis: number; remaining: number } {
    let remaining = quantity
    let costBasis = 0
    while (remaining > 1e-12 && this.head < this.items.length) {
      const item = this.items[this.head]!
      if (item.quantity <= remaining) {
        costBasis += item.quantity * item.price
        remaining -= item.quantity
        this.head++
      } else {
        costBasis += remaining * item.price
        item.quantity -= remaining
        remaining = 0
      }
    }
    return { costBasis, remaining }
  }
}

export interface DailyPnlPoint {
  date: string
  pnl: number
}

export function computeRealizedPnlByDay(trades: BBGoTrade[]): DailyPnlPoint[] {
  if (trades.length === 0) return []

  const bySymbol = new Map<string, BBGoTrade[]>()
  for (const trade of trades) {
    if (!trade.tradedAt) continue
    const list = bySymbol.get(trade.symbol) ?? []
    list.push(trade)
    bySymbol.set(trade.symbol, list)
  }

  const dailyMap = new Map<string, number>()
  for (const [, symTrades] of bySymbol) {
    symTrades.sort((a, b) => (a.tradedAt ?? '').localeCompare(b.tradedAt ?? ''))
    const queue = new FifoQueue()

    for (const trade of symTrades) {
      const day = trade.tradedAt!.slice(0, 10)
      const qty = parseFloat(trade.quantity)
      const price = parseFloat(trade.price)
      const fee = Math.abs(parseFloat(trade.fee || '0'))

      if (trade.side === 'BUY') {
        queue.push(price, qty)
        dailyMap.set(day, (dailyMap.get(day) ?? 0) - fee)
      } else {
        const { costBasis, remaining } = queue.match(qty)
        const adjustedCost = remaining > 0 ? costBasis + remaining * price : costBasis
        const realized = (price * qty) - adjustedCost - fee
        dailyMap.set(day, (dailyMap.get(day) ?? 0) + realized)
      }
    }
  }

  return Array.from(dailyMap.entries())
    .map(([date, pnl]) => ({ date, pnl: Math.round(pnl * 100) / 100 }))
    .sort((a, b) => a.date.localeCompare(b.date))
}

export interface CumulativePnlPoint {
  date: string
  cumulativePnl: number
}

export function computeCumulativePnl(trades: BBGoTrade[]): CumulativePnlPoint[] {
  const daily = computeRealizedPnlByDay(trades)
  let cumulative = 0
  return daily.map(({ date, pnl }) => {
    cumulative += pnl
    return { date, cumulativePnl: cumulative }
  })
}

const QUOTE_CURRENCIES = ['USDT', 'FDUSD', 'BUSD', 'USDC', 'TUSD', 'DAI', 'BTC', 'ETH', 'BNB']
  .sort((a, b) => b.length - a.length)

export function extractBaseCurrency(symbol: string): string {
  for (const q of QUOTE_CURRENCIES) {
    if (symbol.endsWith(q)) return symbol.slice(0, -q.length)
  }
  return symbol
}
