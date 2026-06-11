import type { BBGoOrder, BBGoTrade, TradeMarkersResponse } from './queries'
import type { TradeMarker, OrderLevel } from '@/components/chart/CandlestickChart'

type TradeMarkerAction = TradeMarker['positionAction']

function positionActionToTag(action: string | null | undefined): TradeMarkerAction {
  if (!action) return 'trade'
  const map: Record<string, string> = {
    OPEN: 'open', ADD: 'add', REDUCE: 'reduce', CLOSE: 'close',
    OPEN_LONG: 'openLong', ADD_LONG: 'addLong', REDUCE_LONG: 'reduceLong', CLOSE_LONG: 'closeLong',
    OPEN_SHORT: 'openShort', ADD_SHORT: 'addShort', REDUCE_SHORT: 'reduceShort', CLOSE_SHORT: 'closeShort',
    FLIP_LONG_TO_SHORT: 'flipLongToShort', FLIP_SHORT_TO_LONG: 'flipShortToLong',
  }
  return (map[action] ?? 'trade') as TradeMarkerAction
}

export function buildTradeMarkers(
  trades: BBGoTrade[] | null | undefined,
  _closedOrders: BBGoOrder[] | null | undefined,
  symbol: string,
  _isFutures?: boolean
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
          positionAction: positionActionToTag(tr.positionAction ?? null),
        }))
    )
  }

  return markers.sort((a, b) => (a.time as number) - (b.time as number))
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
