import { describe, it, expect } from 'vitest'
import { buildTradeMarkers, buildOrderLevels } from '../trade-markers'
import type { BBGoOrder, BBGoTrade } from '../queries'

describe('buildTradeMarkers', () => {
  const baseTrade: BBGoTrade = {
    gid: 0,
    id: 1,
    orderID: 100,
    exchange: 'binance',
    symbol: 'BTCUSDT',
    side: 'BUY',
    price: '72000',
    quantity: '0.001',
    tradedAt: '2026-05-28T10:00:00Z',
    fee: '0.01',
    feeCurrency: 'USDT',
    isMaker: true,
    isBuyer: true,
    quoteQuantity: '72',
  }

  const baseOrder: BBGoOrder = {
    gid: 0,
    orderID: 100,
    exchange: 'binance',
    symbol: 'BTCUSDT',
    side: 'SELL',
    price: '73000',
    quantity: '0.001',
    executedQuantity: '0.001',
    orderType: 'LIMIT',
    status: 'Filled',
    creationTime: '2026-05-28T11:00:00Z',
  }

  it('returns empty when no trades or orders', () => {
    expect(buildTradeMarkers(null, null, 'BTCUSDT')).toEqual([])
  })

  it('converts trades to markers', () => {
    const result = buildTradeMarkers([baseTrade], null, 'BTCUSDT')
    expect(result).toHaveLength(1)
    expect(result[0]!.side).toBe('BUY')
    expect(result[0]!.price).toBe(72000)
  })

  it('filters trades by symbol', () => {
    const trade: BBGoTrade = { ...baseTrade, symbol: 'ETHUSDT' }
    expect(buildTradeMarkers([trade], null, 'BTCUSDT')).toHaveLength(0)
  })

  it('includes all symbols when symbol filter is empty', () => {
    const trade: BBGoTrade = { ...baseTrade, symbol: 'ETHUSDT' }
    expect(buildTradeMarkers([trade], null, '')).toHaveLength(1)
  })

  it('converts closed orders with executed quantity > 0 to markers', () => {
    const result = buildTradeMarkers(null, [baseOrder], 'BTCUSDT')
    expect(result).toHaveLength(1)
    expect(result[0]!.side).toBe('SELL')
    expect(result[0]!.price).toBe(73000)
  })

  it('skips orders with zero executed quantity', () => {
    const order: BBGoOrder = { ...baseOrder, executedQuantity: '0' }
    expect(buildTradeMarkers(null, [order], 'BTCUSDT')).toHaveLength(0)
  })

  it('merges trades and orders, sorted by time', () => {
    const result = buildTradeMarkers([baseTrade], [baseOrder], 'BTCUSDT')
    expect(result).toHaveLength(2)
    expect(result[0]!.price).toBe(72000)
    expect(result[1]!.price).toBe(73000)
  })

  it('deduplicates markers with same time+side+price', () => {
    const order: BBGoOrder = {
      ...baseOrder,
      side: 'BUY',
      price: '72000',
      executedQuantity: '0.001',
      creationTime: '2026-05-28T10:00:00Z',
    }
    const result = buildTradeMarkers([baseTrade], [order], 'BTCUSDT')
    expect(result).toHaveLength(1)
  })

  it('keeps markers with same price but different side', () => {
    const order: BBGoOrder = {
      ...baseOrder,
      side: 'SELL',
      price: '72000',
      executedQuantity: '0.001',
      creationTime: '2026-05-28T10:00:00Z',
    }
    const result = buildTradeMarkers([baseTrade], [order], 'BTCUSDT')
    expect(result).toHaveLength(2)
  })

  it('limits to 200 trade markers', () => {
    const trades = Array.from({ length: 250 }, (_, i) => ({
      ...baseTrade,
      id: i,
      price: String(70000 + i),
      tradedAt: new Date(Date.parse('2026-05-28T10:00:00Z') + i * 60000).toISOString(),
    }))
    const result = buildTradeMarkers(trades, null, 'BTCUSDT')
    expect(result).toHaveLength(200)
  })

  it('handles orders with missing creationTime', () => {
    const order: BBGoOrder = { ...baseOrder, creationTime: undefined }
    const result = buildTradeMarkers(null, [order], 'BTCUSDT')
    expect(result).toHaveLength(1)
    expect(result[0]!.time).toBeGreaterThan(0)
  })
})

describe('buildOrderLevels', () => {
  const baseOrder: BBGoOrder = {
    gid: 0,
    orderID: 100,
    exchange: 'binance',
    symbol: 'BTCUSDT',
    side: 'BUY',
    price: '72000',
    quantity: '0.001',
    executedQuantity: '0',
    orderType: 'LIMIT',
    status: 'New',
  }

  it('returns empty for null orders', () => {
    expect(buildOrderLevels(null, 'BTCUSDT')).toEqual([])
  })

  it('converts orders to levels', () => {
    const result = buildOrderLevels([baseOrder], 'BTCUSDT')
    expect(result).toHaveLength(1)
    expect(result[0]!.price).toBe(72000)
    expect(result[0]!.side).toBe('BUY')
  })

  it('filters by symbol', () => {
    const order: BBGoOrder = { ...baseOrder, symbol: 'ETHUSDT' }
    expect(buildOrderLevels([order], 'BTCUSDT')).toHaveLength(0)
  })

  it('uses executedQuantity if present, falls back to quantity', () => {
    const order: BBGoOrder = { ...baseOrder, executedQuantity: '0.5' }
    const result = buildOrderLevels([order], 'BTCUSDT')
    expect(result[0]!.quantity).toBe('0.5')
  })

  it('falls back to quantity when executedQuantity is empty', () => {
    const order: BBGoOrder = { ...baseOrder, executedQuantity: '' }
    const result = buildOrderLevels([order], 'BTCUSDT')
    expect(result[0]!.quantity).toBe('0.001')
  })
})
