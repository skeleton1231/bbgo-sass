import type { BBGoOrder, BBGoTrade } from './queries'
import type { TradeMarker, OrderLevel } from '@/components/chart/CandlestickChart'

export function buildTradeMarkers(
  trades: BBGoTrade[] | null | undefined,
  closedOrders: BBGoOrder[] | null | undefined,
  symbol: string
): TradeMarker[] {
  const markers: TradeMarker[] = []

  if (trades) {
    markers.push(
      ...trades
        .filter((tr) => !symbol || tr.symbol === symbol)
        .slice(0, 200)
        .map((tr) => ({
          time: Math.floor(new Date(tr.tradedAt).getTime() / 1000) as TradeMarker['time'],
          side: tr.side as 'BUY' | 'SELL',
          price: parseFloat(tr.price),
          quantity: parseFloat(tr.quantity),
        }))
    )
  }

  if (closedOrders) {
    markers.push(
      ...closedOrders
        .filter((o) => (!symbol || o.symbol === symbol) && parseFloat(o.executedQuantity) > 0)
        .map((o) => ({
          time: Math.floor(new Date(o.creationTime ?? Date.now()).getTime() / 1000) as TradeMarker['time'],
          side: o.side as 'BUY' | 'SELL',
          price: parseFloat(o.price),
          quantity: parseFloat(o.executedQuantity || o.quantity),
          orderId: o.orderID,
        }))
    )
  }

  const seen = new Set<string>()
  return markers
    .sort((a, b) => (a.time as number) - (b.time as number))
    .filter((m) => {
      const key = `${m.time}-${m.side}-${m.price}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
}

export function buildOrderLevels(
  orders: BBGoOrder[] | null | undefined,
  symbol: string
): OrderLevel[] {
  if (!orders) return []
  return orders
    .filter((o) => !symbol || o.symbol === symbol)
    .map((o) => ({
      price: parseFloat(o.price),
      side: o.side as 'BUY' | 'SELL',
      quantity: o.executedQuantity || o.quantity,
    }))
}
