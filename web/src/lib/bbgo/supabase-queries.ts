import { useQuery } from '@tanstack/react-query'
import { createClient } from '@/lib/supabase/client'
import type { Database } from '@/lib/supabase/types'
import type { BBGoTrade, BBGoOrder, PnLReport, DailyPnl, PnlCurvePoint, TradingVolumeEntry } from './queries'
import { FifoQueue } from './fifo-pnl'
import { tableName, tradeRowToBBGo, orderRowToBBGo } from './supabase-adapters'

export { tableName, tradeRowToBBGo, orderRowToBBGo } from './supabase-adapters'

type Tables = Database['public']['Tables']
type OrderRow = Tables['orders']['Row']
type TradeRow = Tables['trades']['Row']
type PositionRow = Tables['positions']['Row']
type ProfitRow = Tables['profits']['Row']

// --- Supabase hooks returning BBGo types ---

export function useSupabaseTrades(
  userId: string,
  opts?: {
    exchange?: string
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
    limit?: number
  }
) {
  return useQuery<BBGoTrade[]>({
    queryKey: ['supabase-trades', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      const tbl = tableName('trades', opts?.mode)
      let q = sb
        .from(tbl)
        .select('*')
        .eq('user_id', userId)
        .order('traded_at', { ascending: false })
        .limit(opts?.limit ?? 200)

      if (opts?.exchange) q = q.eq('exchange', opts.exchange)
      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q
      if (error) throw error
      return (data ?? []).map((row, i) => tradeRowToBBGo(row as TradeRow, i))
    },
    enabled: !!userId,
    staleTime: 15_000,
  })
}

export function useSupabaseClosedOrders(
  userId: string,
  opts?: {
    exchange?: string
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
    limit?: number
  }
) {
  return useQuery<BBGoOrder[]>({
    queryKey: ['supabase-closed-orders', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      let q = sb
        .from(tableName('orders', opts?.mode))
        .select('*')
        .eq('user_id', userId)
        .neq('status', 'NEW')
        .neq('status', 'PARTIALLY_FILLED')
        .order('created_at', { ascending: false })
        .limit(opts?.limit ?? 200)

      if (opts?.exchange) q = q.eq('exchange', opts.exchange)
      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q
      if (error) throw error
      return (data ?? []).map((row, i) => orderRowToBBGo(row as OrderRow, i))
    },
    enabled: !!userId,
    staleTime: 15_000,
  })
}

export function useSupabasePositions(
  userId: string,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  return useQuery<PositionRow[]>({
    queryKey: ['supabase-positions', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      let q = sb
        .from(tableName('positions', opts?.mode))
        .select('*')
        .eq('user_id', userId)
        .order('traded_at', { ascending: false })

      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q
      if (error) throw error
      return (data ?? []) as PositionRow[]
    },
    enabled: !!userId,
    staleTime: 15_000,
  })
}

export function useSupabaseProfits(
  userId: string,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
    limit?: number
  }
) {
  return useQuery<ProfitRow[]>({
    queryKey: ['supabase-profits', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      let q = sb
        .from(tableName('profits', opts?.mode))
        .select('*')
        .eq('user_id', userId)
        .order('traded_at', { ascending: false })
        .limit(opts?.limit ?? 500)

      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q
      if (error) throw error
      return (data ?? []) as ProfitRow[]
    },
    enabled: !!userId,
    staleTime: 15_000,
  })
}

// --- PnL from profits table (bbgo average-cost method) ---

export interface ProfitAggregation {
  totalNetProfit: number
  totalGrossProfit: number
  totalGrossLoss: number
  totalFees: number
  profitCount: number
  lossCount: number
  totalTrades: number
  winRate: number
  profitFactor: number
  pnlCurve: PnlCurvePoint[]
  dailyBreakdown: DailyPnl[]
}

function aggregateProfits(rows: ProfitRow[]): ProfitAggregation {
  let totalNetProfit = 0
  let totalGrossProfit = 0
  let totalGrossLoss = 0
  let totalFees = 0
  let profitCount = 0
  let lossCount = 0

  const dailyMap = new Map<string, { netProfit: number; fees: number }>()

  // Sort ascending for cumulative curve
  const sorted = [...rows].sort(
    (a, b) => a.traded_at.localeCompare(b.traded_at)
  )

  let cumulative = 0
  const pnlCurve: PnlCurvePoint[] = []

  for (const row of sorted) {
    const netProfit = parseFloat(row.net_profit)
    const fee = Math.abs(parseFloat(row.fee))

    totalNetProfit += netProfit
    totalFees += fee

    if (netProfit > 0) {
      profitCount++
      totalGrossProfit += netProfit
    } else if (netProfit < 0) {
      lossCount++
      totalGrossLoss += netProfit
    }

    // Daily breakdown
    const day = row.traded_at.slice(0, 10)
    const dayEntry = dailyMap.get(day) ?? { netProfit: 0, fees: 0 }
    dayEntry.netProfit += netProfit
    dayEntry.fees += fee
    dailyMap.set(day, dayEntry)

    // Cumulative PnL curve
    cumulative += netProfit
    const ts = Math.floor(new Date(day + 'T00:00:00Z').getTime() / 1000)
    if (pnlCurve.length === 0 || pnlCurve[pnlCurve.length - 1]!.time !== ts) {
      pnlCurve.push({ time: ts, value: Math.round(cumulative * 100) / 100 })
    } else {
      pnlCurve[pnlCurve.length - 1]!.value = Math.round(cumulative * 100) / 100
    }
  }

  const totalTrades = profitCount + lossCount
  const dailyEntries = Array.from(dailyMap.entries()).sort(([a], [b]) => a.localeCompare(b))
  let runningPnl = 0
  const dailyBreakdown: DailyPnl[] = []
  for (const [date, entry] of dailyEntries) {
    runningPnl += entry.netProfit
    dailyBreakdown.push({
      date,
      pnl: Math.round(runningPnl * 100) / 100,
      realizedPnl: Math.round(entry.netProfit * 100) / 100,
      fees: Math.round(entry.fees * 100) / 100,
      trades: 0,
    } as DailyPnl)
  }

  return {
    totalNetProfit: Math.round(totalNetProfit * 1e6) / 1e6,
    totalGrossProfit: Math.round(totalGrossProfit * 1e6) / 1e6,
    totalGrossLoss: Math.round(totalGrossLoss * 1e6) / 1e6,
    totalFees: Math.round(totalFees * 1e6) / 1e6,
    profitCount,
    lossCount,
    totalTrades,
    winRate: totalTrades > 0 ? Math.round((profitCount / totalTrades) * 1000) / 10 : 0,
    profitFactor: totalGrossLoss !== 0
      ? Math.round((totalGrossProfit / Math.abs(totalGrossLoss)) * 100) / 100
      : totalGrossProfit > 0 ? Infinity : 0,
    pnlCurve,
    dailyBreakdown,
  }
}

export function useSupabasePnLFromProfits(
  userId: string,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  const { data: profitRows } = useSupabaseProfits(userId, {
    ...opts,
    limit: 2000,
  })

  return useQuery<ProfitAggregation>({
    queryKey: ['supabase-pnl-profits', userId, opts, profitRows],
    queryFn: () => aggregateProfits(profitRows ?? []),
    enabled: !!userId && !!profitRows,
    staleTime: 30_000,
  })
}

// --- Latest position from positions table ---

export interface LatestPosition {
  symbol: string
  base: number
  quote: number
  averageCost: number
  accumulatedProfit: number
  strategy: string
  strategyInstanceId: string
  exchange: string
  tradedAt: string
  isLong: boolean
  isShort: boolean
  isClosed: boolean
}

export function useSupabaseLatestPosition(
  userId: string,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  const { data: positions } = useSupabasePositions(userId, opts)

  return useQuery<LatestPosition | null>({
    queryKey: ['supabase-latest-position', userId, opts, positions],
    queryFn: () => {
      if (!positions || positions.length === 0) return null
      const latest = positions[0]!
      const base = parseFloat(latest.base)
      return {
        symbol: latest.symbol,
        base,
        quote: parseFloat(latest.quote),
        averageCost: parseFloat(latest.average_cost),
        accumulatedProfit: parseFloat(latest.profit ?? '0'),
        strategy: latest.strategy,
        strategyInstanceId: latest.strategy_instance_id,
        exchange: latest.exchange,
        tradedAt: latest.traded_at,
        isLong: base > 0,
        isShort: base < 0,
        isClosed: base === 0,
      }
    },
    enabled: !!userId && !!positions,
    staleTime: 15_000,
  })
}

// --- Unrealized PnL from position + current price (matches bbgo Position.UnrealizedProfit) ---

export function useUnrealizedPnL(
  userId: string,
  currentPrice: number | undefined,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  const { data: position } = useSupabaseLatestPosition(userId, opts)

  return useQuery<{ unrealizedPnl: number; unrealizedPnlPct: number }>({
    queryKey: ['unrealized-pnl', userId, opts, position, currentPrice],
    queryFn: () => {
      if (!position || position.isClosed || !currentPrice || currentPrice === 0) {
        return { unrealizedPnl: 0, unrealizedPnlPct: 0 }
      }
      // Matches bbgo Position.unrealizedProfit():
      // Long:  (price - averageCost) * |base|
      // Short: (averageCost - price) * |base|
      const absBase = Math.abs(position.base)
      let unrealizedPnl: number
      if (position.isLong) {
        unrealizedPnl = (currentPrice - position.averageCost) * absBase
      } else {
        unrealizedPnl = (position.averageCost - currentPrice) * absBase
      }
      const cost = position.averageCost * absBase
      const unrealizedPnlPct = cost > 0 ? (unrealizedPnl / cost) * 100 : 0

      return {
        unrealizedPnl: Math.round(unrealizedPnl * 1e6) / 1e6,
        unrealizedPnlPct: Math.round(unrealizedPnlPct * 100) / 100,
      }
    },
    enabled: !!userId && !!position,
    staleTime: 15_000,
  })
}

// --- Legacy PnL computed from trades (fallback when profits table is empty) ---

export function useSupabasePnL(
  userId: string,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  const { data: trades } = useSupabaseTrades(userId, {
    ...opts,
    limit: 1000,
  })

  return useQuery<PnLReport>({
    queryKey: ['supabase-pnl', userId, opts, trades],
    queryFn: () => computePnLFromTrades(trades ?? []),
    enabled: !!userId && !!trades,
    staleTime: 30_000,
  })
}

function computePnLFromTrades(trades: BBGoTrade[]): PnLReport {
  if (trades.length === 0) {
    return {
      totalRealizedPnl: 0, totalUnrealizedPnl: 0, totalFees: 0,
      totalTrades: 0, winningTrades: 0, losingTrades: 0, winRate: 0,
      symbols: [], dailyBreakdown: [], pnlCurve: [],
    }
  }

  const sorted = [...trades].sort(
    (a, b) => (a.tradedAt ?? '').localeCompare(b.tradedAt ?? '')
  )

  const bySymbol = new Map<string, BBGoTrade[]>()
  for (const t of sorted) {
    const list = bySymbol.get(t.symbol) ?? []
    list.push(t)
    bySymbol.set(t.symbol, list)
  }

  let totalRealized = 0
  let totalFees = 0
  let totalTrades = 0
  let winning = 0
  let losing = 0
  const symbols: PnLReport['symbols'] = []
  const dailyMap = new Map<string, { realized: number; fees: number }>()

  for (const [symbol, symTrades] of bySymbol) {
    const queue = new FifoQueue()
    let symRealized = 0
    let symFees = 0
    let buys = 0
    let sells = 0
    let buyVol = 0
    let sellVol = 0
    let symWin = 0
    let symLose = 0
    let symCount = 0
    let openQty = 0
    let openCost = 0

    for (const trade of symTrades) {
      const qty = parseFloat(trade.quantity)
      const price = parseFloat(trade.price)
      const fee = Math.abs(parseFloat(trade.fee || '0'))

      symFees += fee
      symCount++

      const day = trade.tradedAt?.slice(0, 10) ?? ''
      const dayEntry = dailyMap.get(day) ?? { realized: 0, fees: 0 }
      dayEntry.fees += fee

      if (trade.side === 'BUY') {
        queue.push(price, qty)
        buys++
        buyVol += qty * price
        openQty += qty
        openCost += qty * price
      } else {
        const { costBasis, remaining } = queue.match(qty)
        let matchedCost = costBasis
        if (remaining > 0) matchedCost += remaining * price
        const realized = price * qty - matchedCost
        symRealized += realized
        dayEntry.realized += realized
        sells++
        sellVol += qty * price
        openQty -= qty
        openCost -= matchedCost
        if (openQty < 0) openQty = 0
        if (realized > 0) symWin++
        else if (realized < 0) symLose++
      }

      dailyMap.set(day, dayEntry)
    }

    totalRealized += symRealized
    totalFees += symFees
    totalTrades += symCount
    winning += symWin
    losing += symLose

    symbols.push({
      symbol,
      realizedPnl: Math.round(symRealized * 1e6) / 1e6,
      totalBuys: buys,
      totalSells: sells,
      buyVolume: buyVol,
      sellVolume: sellVol,
      totalFees: Math.round(symFees * 1e6) / 1e6,
      tradeCount: symCount,
      winningTrades: symWin,
      losingTrades: symLose,
      avgBuyPrice: buys > 0 ? buyVol / buys : 0,
      avgSellPrice: sells > 0 ? sellVol / sells : 0,
      openPosition: openQty,
      openPositionCost: openCost,
      unrealizedPnl: 0,
      currentPrice: 0,
    })
  }

  const dailyEntries = Array.from(dailyMap.entries()).sort(([a], [b]) => a.localeCompare(b))
  let runningPnl = 0
  const dailyBreakdown: DailyPnl[] = []
  const pnlCurve: PnlCurvePoint[] = []
  for (const [date, entry] of dailyEntries) {
    runningPnl += entry.realized - entry.fees
    dailyBreakdown.push({
      date,
      pnl: Math.round(runningPnl * 100) / 100,
      realizedPnl: Math.round(entry.realized * 100) / 100,
      fees: Math.round(entry.fees * 100) / 100,
      trades: 0,
    } as DailyPnl)
    const ts = Math.floor(new Date(date + 'T00:00:00Z').getTime() / 1000)
    pnlCurve.push({ time: ts, value: Math.round(runningPnl * 100) / 100 })
  }

  return {
    totalRealizedPnl: Math.round(totalRealized * 1e6) / 1e6,
    totalUnrealizedPnl: 0,
    totalFees: Math.round(totalFees * 1e6) / 1e6,
    totalTrades,
    winningTrades: winning,
    losingTrades: losing,
    winRate: totalTrades > 0 ? Math.round((winning / totalTrades) * 1000) / 10 : 0,
    symbols,
    dailyBreakdown,
    pnlCurve,
  }
}

export function useSupabaseTradeCount(
  userId: string,
  opts?: {
    symbol?: string
    mode?: 'live' | 'paper'
  }
) {
  return useQuery<number>({
    queryKey: ['supabase-trade-count', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      let q = sb
        .from(tableName('trades', opts?.mode))
        .select('id', { count: 'exact', head: true })
        .eq('user_id', userId)

      if (opts?.symbol) q = q.eq('symbol', opts.symbol)

      const { count, error } = await q
      if (error) throw error
      return count ?? 0
    },
    enabled: !!userId,
    staleTime: 30_000,
  })
}

export function useSupabaseTradingVolume(
  userId: string,
  opts?: { mode?: 'live' | 'paper'; period?: string }
) {
  const { data: trades } = useSupabaseTrades(userId, { mode: opts?.mode, limit: 1000 })

  return useQuery<{ tradingVolumes: TradingVolumeEntry[] }>({
    queryKey: ['supabase-trading-volume', userId, opts, trades],
    queryFn: () => {
      const period = opts?.period ?? '30d'
      const byDay = new Map<string, number>()
      const now = new Date()

      for (const t of trades ?? []) {
        if (!t.tradedAt) continue
        const qty = parseFloat(t.quantity)
        const price = parseFloat(t.price)
        const vol = qty * price
        const day = t.tradedAt.slice(0, 10)
        byDay.set(day, (byDay.get(day) ?? 0) + vol)
      }

      let days = 30
      if (period === '7d') days = 7
      else if (period === 'year') days = 365
      else if (period === 'month') days = 30

      const volumes: TradingVolumeEntry[] = []
      for (const [date, vol] of byDay) {
        const d = new Date(date)
        if (now.getTime() - d.getTime() > days * 86_400_000) continue
        volumes.push({ year: d.getFullYear(), month: d.getMonth() + 1, day: d.getDate(), quoteVolume: Math.round(vol * 100) / 100 })
      }
      volumes.sort((a, b) => (a.year - b.year) || ((a.month ?? 0) - (b.month ?? 0)) || ((a.day ?? 0) - (b.day ?? 0)))
      return { tradingVolumes: volumes }
    },
    enabled: !!userId && !!trades,
    staleTime: 60_000,
  })
}
