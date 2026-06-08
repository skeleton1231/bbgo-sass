import { describe, it, expect } from 'vitest'
import { tradeRowToBBGo, orderRowToBBGo, tableName } from '../supabase-adapters'

describe('tableName', () => {
  it('returns base table for live mode', () => {
    expect(tableName('orders')).toBe('orders')
    expect(tableName('trades', 'live')).toBe('trades')
  })

  it('prefixes paper_ for paper mode', () => {
    expect(tableName('orders', 'paper')).toBe('paper_orders')
    expect(tableName('trades', 'paper')).toBe('paper_trades')
    expect(tableName('positions', 'paper')).toBe('paper_positions')
    expect(tableName('profits', 'paper')).toBe('paper_profits')
  })
})

describe('tradeRowToBBGo', () => {
  const row = {
    trade_id: '999',
    order_id: '12345',
    order_uuid: 'order-uuid-1',
    symbol: 'BTCUSDT',
    side: 'BUY',
    price: '50000',
    quantity: '0.1',
    quote_quantity: '5000',
    fee: '0.001',
    fee_currency: 'USDT',
    exchange: 'binance',
    is_buyer: true,
    is_maker: false,
    traded_at: '2026-01-15T10:30:00Z',
  }

  it('maps all fields correctly', () => {
    const trade = tradeRowToBBGo(row, 0)
    expect(trade.id).toBe(999)
    expect(trade.orderID).toBe(12345)
    expect(trade.orderUUID).toBe('order-uuid-1')
    expect(trade.symbol).toBe('BTCUSDT')
    expect(trade.side).toBe('BUY')
    expect(trade.price).toBe('50000')
    expect(trade.quantity).toBe('0.1')
    expect(trade.quoteQuantity).toBe('5000')
    expect(trade.fee).toBe('0.001')
    expect(trade.feeCurrency).toBe('USDT')
    expect(trade.exchange).toBe('binance')
    expect(trade.isBuyer).toBe(true)
    expect(trade.isMaker).toBe(false)
    expect(trade.tradedAt).toBe('2026-01-15T10:30:00Z')
    expect(trade.gid).toBe(0)
  })

  it('uses index as fallback for invalid trade_id', () => {
    const result = tradeRowToBBGo({ ...row, trade_id: 'abc' }, 5)
    expect(result.id).toBe(5)
  })

  it('handles missing optional fields', () => {
    const minimal = { ...row, order_uuid: '', quote_quantity: null }
    const trade = tradeRowToBBGo(minimal, 0)
    expect(trade.orderUUID).toBeUndefined()
    expect(trade.quoteQuantity).toBe('0')
  })
})

describe('orderRowToBBGo', () => {
  const row = {
    order_id: '56789',
    order_uuid: 'order-uuid-2',
    client_order_id: 'client-abc',
    symbol: 'ETHUSDT',
    side: 'SELL',
    order_type: 'limit',
    price: '3000',
    quantity: '1.5',
    executed_quantity: '1.5',
    status: 'FILLED',
    stop_price: '0',
    is_working: true,
    created_at: '2026-01-15T12:00:00Z',
  }

  it('maps all fields correctly', () => {
    const order = orderRowToBBGo(row, 0)
    expect(order.orderID).toBe(56789)
    expect(order.uuid).toBe('order-uuid-2')
    expect(order.clientOrderID).toBe('client-abc')
    expect(order.symbol).toBe('ETHUSDT')
    expect(order.side).toBe('SELL')
    expect(order.orderType).toBe('limit')
    expect(order.price).toBe('3000')
    expect(order.quantity).toBe('1.5')
    expect(order.executedQuantity).toBe('1.5')
    expect(order.status).toBe('FILLED')
    expect(order.isWorking).toBe(true)
    expect(order.creationTime).toBe('2026-01-15T12:00:00Z')
  })

  it('omits stopPrice when zero', () => {
    const order = orderRowToBBGo(row, 0)
    expect(order.stopPrice).toBeUndefined()
  })

  it('includes stopPrice when non-zero', () => {
    const order = orderRowToBBGo({ ...row, stop_price: '2900' }, 0)
    expect(order.stopPrice).toBe('2900')
  })

  it('handles missing optional fields', () => {
    const minimal = { ...row, order_uuid: '', client_order_id: '' }
    const order = orderRowToBBGo(minimal, 3)
    expect(order.uuid).toBeUndefined()
    expect(order.clientOrderID).toBeUndefined()
    expect(order.gid).toBe(3)
  })
})
