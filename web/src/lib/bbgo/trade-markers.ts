import type { BBGoOrder, BBGoTrade, TradeMarkersResponse } from './queries'
import type { TradeMarker, OrderLevel } from '@/components/chart/CandlestickChart'
import { computePositionTags, computeFuturesPositionTags } from './position-tags'

export function buildTradeMarkers(
  trades: BBGoTrade[] | null | undefined,
  _closedOrders: BBGoOrder[] | null | undefined,
  symbol: string,
  isFutures?: boolean
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

  const sorted = markers.sort((a, b) => (a.time as number) - (b.time as number))

  if (isFutures) {
    const tags = computeFuturesPositionTags(sorted.map((m) => ({ side: m.side, quantity: String(m.quantity), tradedAt: String(m.time) })))
    for (let i = 0; i < sorted.length; i++) {
      sorted[i]!.positionAction = tags[i]!.tag ?? 'trade'
    }
  } else {
    const tags = computePositionTags(sorted.map((m) => ({ side: m.side, quantity: String(m.quantity), tradedAt: String(m.time) })))
    for (let i = 0; i < sorted.length; i++) {
      sorted[i]!.positionAction = tags[i]!.tag ?? 'trade'
    }
  }

  return sorted
}

export function buildTradeMarkersFromServer(data: TradeMarkersResponse | undefined): TradeMarker[] {
  if (!data?.markers?.length) return null as unknown as TradeMarker[]
  return data.markers.map((m) => ({
    time: m.time as TradeMarker['time'],
    side: m.side as 'BUY' | 'SELL',
    price: m.price,
    quantity: m.quantity,
    positionAction: (m.positionAction || 'trade') as TradeMarker['positionAction'],
  }))
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
      quantity: o.quantity,
    }))
}
