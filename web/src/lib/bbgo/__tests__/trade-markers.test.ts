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

  it('ignores closed orders (uses only trades for accurate fill times)', () => {
    const result = buildTradeMarkers(null, [baseOrder], 'BTCUSDT')
    expect(result).toHaveLength(0)
  })

  it('uses only trades even when closed orders are provided', () => {
    const result = buildTradeMarkers([baseTrade], [baseOrder], 'BTCUSDT')
    expect(result).toHaveLength(1)
    expect(result[0]!.side).toBe('BUY')
    expect(result[0]!.price).toBe(72000)
  })

  it('deduplicates trades with same time+side+price', () => {
    const dup: BBGoTrade = { ...baseTrade }
    const result = buildTradeMarkers([baseTrade, dup], null, 'BTCUSDT')
    expect(result).toHaveLength(1)
  })

  it('keeps trades with same price but different side', () => {
    const sell: BBGoTrade = { ...baseTrade, side: 'SELL', tradedAt: '2026-05-28T10:30:00Z' }
    const result = buildTradeMarkers([baseTrade, sell], null, 'BTCUSDT')
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

  it('tags first BUY as open, last SELL as close', () => {
    const buy1: BBGoTrade = { ...baseTrade, side: 'BUY', quantity: '0.001', price: '72000', tradedAt: '2026-05-28T10:00:00Z' }
    const buy2: BBGoTrade = { ...baseTrade, side: 'BUY', quantity: '0.001', price: '72200', tradedAt: '2026-05-28T10:30:00Z' }
    const sell1: BBGoTrade = { ...baseTrade, side: 'SELL', quantity: '0.001', price: '72800', tradedAt: '2026-05-28T11:00:00Z' }
    const sell2: BBGoTrade = { ...baseTrade, side: 'SELL', quantity: '0.001', price: '73000', tradedAt: '2026-05-28T11:30:00Z' }
    const result = buildTradeMarkers([buy1, buy2, sell1, sell2], null, 'BTCUSDT')
    expect(result[0]!.positionAction).toBe('open')
    expect(result[1]!.positionAction).toBe('add')
    expect(result[2]!.positionAction).toBe('reduce')
    expect(result[3]!.positionAction).toBe('close')
  })

  it('tags single BUY then single SELL as open then close', () => {
    const buy: BBGoTrade = { ...baseTrade, side: 'BUY', quantity: '0.001', price: '72000', tradedAt: '2026-05-28T10:00:00Z' }
    const sell: BBGoTrade = { ...baseTrade, side: 'SELL', quantity: '0.001', price: '73000', tradedAt: '2026-05-28T11:00:00Z' }
    const result = buildTradeMarkers([buy, sell], null, 'BTCUSDT')
    expect(result[0]!.positionAction).toBe('open')
    expect(result[1]!.positionAction).toBe('close')
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

  it('uses order quantity for chart labels', () => {
    const order: BBGoOrder = { ...baseOrder, executedQuantity: '0.5' }
    const result = buildOrderLevels([order], 'BTCUSDT')
    expect(result[0]!.quantity).toBe('0.001')
  })
})
