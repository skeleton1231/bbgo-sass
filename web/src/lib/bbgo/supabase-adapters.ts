import type { BBGoTrade, BBGoOrder } from './queries'

export function tableName(base: string, mode?: 'live' | 'paper'): string {
  return mode === 'paper' ? `paper_${base}` : base
}

export function tradeRowToBBGo(row: Record<string, unknown>, idx: number): BBGoTrade {
  const strategyInstanceId = (row.strategy_instance_id as string) || undefined
  return {
    gid: idx,
    id: parseInt(String(row.trade_id ?? idx), 10) || idx,
    orderID: parseInt(String(row.order_id ?? 0), 10) || 0,
    orderUUID: (row.order_uuid as string) || undefined,
    exchange: String(row.exchange ?? ''),
    symbol: String(row.symbol ?? ''),
    side: String(row.side ?? '') as 'BUY' | 'SELL',
    price: String(row.price ?? '0'),
    quantity: String(row.quantity ?? '0'),
    quoteQuantity: String(row.quote_quantity ?? '0'),
    isBuyer: Boolean(row.is_buyer),
    isMaker: Boolean(row.is_maker),
    tradedAt: String(row.traded_at ?? ''),
    fee: String(row.fee ?? '0'),
    feeCurrency: String(row.fee_currency ?? ''),
    strategyInstanceId,
  }
}

export function orderRowToBBGo(row: Record<string, unknown>, idx: number): BBGoOrder {
  const stopPrice = String(row.stop_price ?? '0')
  const strategyInstanceId = (row.strategy_instance_id as string) || undefined
  return {
    gid: idx,
    orderID: parseInt(String(row.order_id ?? idx), 10) || idx,
    uuid: (row.order_uuid as string) || undefined,
    clientOrderID: (row.client_order_id as string) || undefined,
    exchange: String(row.exchange ?? ''),
    symbol: String(row.symbol ?? ''),
    side: String(row.side ?? '') as 'BUY' | 'SELL',
    orderType: String(row.order_type ?? ''),
    price: String(row.price ?? '0'),
    quantity: String(row.quantity ?? '0'),
    executedQuantity: String(row.executed_quantity ?? '0'),
    status: String(row.status ?? ''),
    stopPrice: stopPrice !== '0' ? stopPrice : undefined,
    creationTime: String(row.created_at ?? ''),
    isWorking: Boolean(row.is_working),
    tag: (row.tag as string) || undefined,
    strategyInstanceId,
  }
}
