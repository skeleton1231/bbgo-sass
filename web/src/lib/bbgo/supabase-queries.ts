import { useQuery } from '@tanstack/react-query'
import { createClient } from '@/lib/supabase/client'
import type { Database } from '@/lib/supabase/types'
import type { BBGoTrade, BBGoOrder, BBGoBalance, PnLReport, DailyPnl, PnlCurvePoint, TradingVolumeEntry, FuturesPositionRisk, MarginLoan, MarginRepay, MarginInterest, MarginLiquidation } from './queries'
import { FifoQueue } from './fifo-pnl'
import { tableName, tradeRowToBBGo, orderRowToBBGo } from './supabase-adapters'

export { tableName, tradeRowToBBGo, orderRowToBBGo, tradeKey, orderKey } from './supabase-adapters'

type Tables = Database['public']['Tables']
type OrderRow = Tables['orders']['Row']
type TradeRow = Tables['trades']['Row']
type PositionRow = Tables['positions']['Row']
type ProfitRow = Tables['profits']['Row']
type FuturesPositionRiskRow = Tables['futures_position_risks']['Row']
type MarginLoanRow = Tables['margin_loans']['Row']
type MarginRepayRow = Tables['margin_repays']['Row']
type MarginInterestRow = Tables['margin_interests']['Row']
type MarginLiquidationRow = Tables['margin_liquidations']['Row']

// --- Supabase hooks returning BBGo types ---

export function useSupabaseTrades(
  userId: string,
  opts?: {
    exchange?: string
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
    limit?: number
    // 'desc' (default) is for display — newest first.
    // 'asc' is for FIFO PnL computation — needs earliest trades to anchor cost basis.
    order?: 'asc' | 'desc'
    // ISO timestamp; when set, filters `traded_at >= since`. Used to bound PnL
    // computation to a recent window when full history would exceed limit.
    since?: string
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
        .order('traded_at', { ascending: opts?.order !== 'asc' ? false : true })
        .limit(opts?.limit ?? 200)

      if (opts?.exchange) q = q.eq('exchange', opts.exchange)
      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)
      if (opts?.since) q = q.gte('traded_at', opts.since)

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


export function useSupabaseOpenOrders(
  userId: string,
  opts?: {
    exchange?: string
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  return useQuery<BBGoOrder[]>({
    queryKey: ['supabase-open-orders', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      let q = sb
        .from(tableName('orders', opts?.mode))
        .select('*')
        .eq('user_id', userId)
        .eq('status', 'NEW')
        .order('created_at', { ascending: false })

      if (opts?.exchange) q = q.eq('exchange', opts.exchange)
      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q
      if (error) throw error
      return (data ?? []).map((row, i) => orderRowToBBGo(row as OrderRow, i))
    },
    enabled: !!userId,
    staleTime: 10_000,
  })
}

export function useSupabaseBalances(
  userId: string,
  opts?: { mode?: 'live' | 'paper'; strategyInstanceId?: string }
) {
  return useQuery<Record<string, BBGoBalance>>({
    queryKey: ['supabase-balances', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      // Only paper_balances table exists in Supabase; live balances come from exchange.
      if (opts?.mode !== 'paper') return {}

      let q = sb
        .from('paper_balances')
        .select('*')
        .eq('user_id', userId)

      // When filtering by bot, only that bot's row is returned.
      // When unset (dashboard view), all bots' rows are returned and
      // aggregated below — summing available/locked per currency.
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q

      if (error) throw error

      type BalanceRow = { currency: string; available: string; locked: string }
      const balances: Record<string, BBGoBalance> = {}
      for (const row of (data ?? []) as BalanceRow[]) {
        const avail = parseFloat(row.available ?? '0') || 0
        const locked = parseFloat(row.locked ?? '0') || 0
        const existing = balances[row.currency]
        if (existing) {
          existing.available = String((parseFloat(existing.available) || 0) + avail)
          existing.locked = String((parseFloat(existing.locked) || 0) + locked)
        } else {
          balances[row.currency] = {
            currency: row.currency,
            available: String(avail),
            locked: String(locked),
          }
        }
      }
      return balances
    },
    enabled: !!userId,
    staleTime: 10_000,
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
        .order('created_at', { ascending: false })

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

  // Each profit row represents one position event (open with fee-only, or close with realized PnL).
  // totalTrades counts ALL rows so the "Total Trades" stat matches what users see in the trades table.
  // profitCount/lossCount remain restricted to closed positions with non-zero net PnL for win-rate.
  let totalTrades = 0
  const dailyMap = new Map<string, { netProfit: number; fees: number; trades: number }>()

  // Sort ascending for cumulative curve
  const sorted = [...rows].sort(
    (a, b) => a.traded_at.localeCompare(b.traded_at)
  )

  let cumulative = 0
  const pnlCurve: PnlCurvePoint[] = []
  const seenCurveTimes = new Set<number>()

  for (const row of sorted) {
    const netProfit = parseFloat(row.net_profit)
    const fee = Math.abs(parseFloat(row.fee))

    totalNetProfit += netProfit
    totalFees += fee
    totalTrades++

    if (netProfit > 0) {
      profitCount++
      totalGrossProfit += netProfit
    } else if (netProfit < 0) {
      lossCount++
      totalGrossLoss += netProfit
    }

    // Daily breakdown — trades counts every profit-row event
    const day = row.traded_at.slice(0, 10)
    const dayEntry = dailyMap.get(day) ?? { netProfit: 0, fees: 0, trades: 0 }
    dayEntry.netProfit += netProfit
    dayEntry.fees += fee
    dayEntry.trades++
    dailyMap.set(day, dayEntry)

    // Cumulative PnL curve — use real traded_at timestamp to preserve intraday resolution
    cumulative += netProfit
    const ts = Math.floor(new Date(row.traded_at).getTime() / 1000)
    if (!seenCurveTimes.has(ts)) {
      seenCurveTimes.add(ts)
      pnlCurve.push({ time: ts, value: Math.round(cumulative * 100) / 100 })
    } else {
      pnlCurve[pnlCurve.length - 1]!.value = Math.round(cumulative * 100) / 100
    }
  }

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
      trades: entry.trades,
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
  const profitsQuery = useSupabaseProfits(userId, {
    ...opts,
    limit: 2000,
  })
  const profitRows = profitsQuery.data

  const aggregation = useQuery<ProfitAggregation>({
    queryKey: ['supabase-pnl-profits', userId, opts, profitRows],
    queryFn: () => aggregateProfits(profitRows ?? []),
    enabled: !!userId && !!profitRows,
    staleTime: 30_000,
  })

  // Expose the underlying profit rows so consumers (e.g. close-history tab)
  // don't need to fire a second useSupabaseProfits query that overlaps with
  // this one. Different limit → different cache key → duplicate fetch.
  return { ...aggregation, profitRows }
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

function toLatestPosition(row: PositionRow): LatestPosition {
  const base = parseFloat(row.base)
  return {
    symbol: row.symbol,
    base,
    quote: parseFloat(row.quote),
    averageCost: parseFloat(row.average_cost),
    accumulatedProfit: parseFloat(row.profit ?? '0'),
    strategy: row.strategy,
    strategyInstanceId: row.strategy_instance_id,
    exchange: row.exchange,
    tradedAt: row.traded_at,
    isLong: base > 0,
    isShort: base < 0,
    isClosed: base === 0,
  }
}

export function useSupabaseLatestPositions(
  userId: string,
  opts?: {
    symbol?: string
    strategyInstanceId?: string
    mode?: 'live' | 'paper'
  }
) {
  const { data: positions } = useSupabasePositions(userId, opts)

  return useQuery<LatestPosition[]>({
    queryKey: ['supabase-latest-positions', userId, opts, positions],
    queryFn: () => {
      if (!positions || positions.length === 0) return []

      // Group by (exchange, symbol, strategy_instance_id), keep latest per group.
      // strategy_instance_id is necessary so multiple bots on the same symbol don't collapse.
      //
      // Paper engine can emit a zero-base snapshot at the same traded_at as a
      // non-zero one (e.g., liquidation writes both). If we pick the zero row
      // the position looks closed even though it isn't. Guard: when the latest
      // row has base=0 but a sibling row at the same traded_at has non-zero
      // base, prefer the non-zero sibling.
      const latestByGroup = new Map<string, PositionRow>()
      for (const row of positions) {
        const key = `${row.exchange}:${row.symbol}:${row.strategy_instance_id}`
        const current = latestByGroup.get(key)
        if (!current) {
          latestByGroup.set(key, row)
          continue
        }
        const currentBase = parseFloat(current.base ?? '0') || 0
        const rowBase = parseFloat(row.base ?? '0') || 0
        if (currentBase === 0 && rowBase !== 0 && row.traded_at === current.traded_at) {
          latestByGroup.set(key, row)
        }
      }

      return Array.from(latestByGroup.values())
        .map(toLatestPosition)
        .filter((p) => !p.isClosed)
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
  const { data: positions } = useSupabaseLatestPositions(userId, opts)

  return useQuery<{ unrealizedPnl: number; unrealizedPnlPct: number }>({
    queryKey: ['unrealized-pnl', userId, opts, positions, currentPrice],
    queryFn: () => {
      if (!positions || positions.length === 0 || !currentPrice || currentPrice === 0) {
        return { unrealizedPnl: 0, unrealizedPnlPct: 0 }
      }

      let totalPnl = 0
      let totalCost = 0
      for (const pos of positions) {
        if (pos.isClosed) continue
        const absBase = Math.abs(pos.base)
        const pnl = pos.isLong
          ? (currentPrice - pos.averageCost) * absBase
          : (pos.averageCost - currentPrice) * absBase
        totalPnl += pnl
        totalCost += pos.averageCost * absBase
      }

      const unrealizedPnlPct = totalCost > 0 ? (totalPnl / totalCost) * 100 : 0
      return {
        unrealizedPnl: Math.round(totalPnl * 1e6) / 1e6,
        unrealizedPnlPct: Math.round(unrealizedPnlPct * 100) / 100,
      }
    },
    enabled: !!userId && !!positions,
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
    enabled?: boolean
  }
) {
  // ASC order is critical: FIFO cost basis needs the earliest buys to be
  // matched against later sells. DESC would compute PnL against only the
  // most recent 5000 trades' internal matching, hiding long-held positions.
  const enabled = opts?.enabled !== false
  const { data: trades } = useSupabaseTrades(userId, {
    ...opts,
    limit: 5000,
    order: 'asc',
  })

  return useQuery<PnLReport>({
    queryKey: ['supabase-pnl', userId, opts, trades],
    queryFn: () => computePnLFromTrades(trades ?? []),
    enabled: !!userId && !!trades && enabled,
    staleTime: 30_000,
  })
}

function computePnLFromTrades(trades: BBGoTrade[]): PnLReport {
  if (trades.length === 0) {
    return {
      totalRealizedPnl: 0, totalUnrealizedPnl: 0, totalFees: 0,
      totalTrades: 0, winningTrades: 0, losingTrades: 0, winRate: 0,
      profitFactor: 0,
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
    profitFactor: totalRealized < 0
      ? 0
      : totalRealized > 0
        ? Infinity
        : 0,
    symbols,
    dailyBreakdown,
    pnlCurve,
  }
}

export function useSupabaseTradingVolume(
  userId: string,
  opts?: { mode?: 'live' | 'paper'; period?: string }
) {
  // Volume is additive — order doesn't matter, but we cap at 5000 to bound
  // transfer size. For high-frequency bots running >5000 trades in the
  // selected period, volume will be underreported. Server-side aggregation
  // (Supabase RPC summing quote_quantity) is the long-term fix.
  const { data: trades } = useSupabaseTrades(userId, { mode: opts?.mode, limit: 5000 })

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

export function useSupabaseFuturesPositions(
  userId: string,
  opts?: { mode?: 'live' | 'paper'; exchange?: string; symbol?: string; strategyInstanceId?: string }
) {
  return useQuery<FuturesPositionRisk[]>({
    queryKey: ['supabase-futures-positions', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      let q = sb
        .from(tableName('futures_position_risks', opts?.mode))
        .select('*')
        .eq('user_id', userId)
        .order('updated_at', { ascending: false })
        // Cap transfer size. The table is append-only (migration 00036), so without
        // a LIMIT this query pulls the full historical snapshot set on every refetch.
        // Dedup below keeps only the latest per (symbol, side, strategy_instance_id),
        // so 500 is plenty for typical portfolios. The 30s sync throttle (set by
        // manager/instance_store.go) bounds future growth; existing pre-throttle
        // data may exceed this and should be cleaned up via a retention job.
        .limit(500)

      if (opts?.exchange) q = q.eq('exchange', opts.exchange)
      if (opts?.symbol) q = q.eq('symbol', opts.symbol)
      if (opts?.strategyInstanceId) q = q.eq('strategy_instance_id', opts.strategyInstanceId)

      const { data, error } = await q
      if (error) throw error

      // Dedup: keep only the latest snapshot per (exchange, symbol, position_side, strategy_instance_id).
      // In paper one-way mode, position_side is always 'BOTH' (see migration 00046 + paper_trade_futures.go),
      // so this collapses to one row per (symbol, strategy_instance_id). In live hedge mode, Long and Short
      // remain distinct buckets and both positions are preserved.
      const seen = new Set<string>()
      const latest: FuturesPositionRisk[] = []
      for (const row of (data ?? []) as FuturesPositionRisk[]) {
        const key = `${row.exchange}:${row.symbol}:${row.position_side}:${row.strategy_instance_id}`
        if (!seen.has(key)) {
          seen.add(key)
          latest.push(row)
        }
      }
      return latest
    },
    enabled: !!userId,
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export interface FuturesRealtimeMetrics {
  markPrice: number
  unrealizedPnl: number
  unrealizedPnlPct: number
  liqDistancePct: number
  isLive: boolean
}

function num(s: string | undefined | null): number {
  if (!s) return 0
  const v = parseFloat(s)
  return isNaN(v) ? 0 : v
}

export function computeFuturesRealtime(
  risk: FuturesPositionRisk,
  currentPrice: number | undefined,
): FuturesRealtimeMetrics {
  const amount = num(risk.position_amount)
  const entry = num(risk.entry_price)
  const dbMark = num(risk.mark_price)
  const liq = num(risk.liquidation_price)

  const isLive = typeof currentPrice === 'number' && currentPrice > 0 && entry > 0 && amount !== 0
  const markPrice = isLive ? currentPrice! : dbMark

  let unrealizedPnl = 0
  if (entry > 0 && amount !== 0 && markPrice > 0) {
    unrealizedPnl = amount > 0
      ? (markPrice - entry) * amount
      : (entry - markPrice) * Math.abs(amount)
  }
  const cost = entry * Math.abs(amount)
  const unrealizedPnlPct = cost > 0 ? (unrealizedPnl / cost) * 100 : 0

  const liqDistancePct = liq > 0 && markPrice > 0
    ? (Math.abs(markPrice - liq) / markPrice) * 100
    : 0

  return {
    markPrice,
    unrealizedPnl: Math.round(unrealizedPnl * 1e6) / 1e6,
    unrealizedPnlPct: Math.round(unrealizedPnlPct * 100) / 100,
    liqDistancePct: Math.round(liqDistancePct * 100) / 100,
    isLive,
  }
}

export function useSupabaseMarginHistory(
  userId: string,
  opts?: { mode?: 'live' | 'paper' }
) {
  return useQuery<{ loans: MarginLoan[]; repays: MarginRepay[]; interests: MarginInterest[]; liquidations: MarginLiquidation[] }>({
    queryKey: ['supabase-margin-history', userId, opts],
    queryFn: async () => {
      const sb = createClient()
      const mode = opts?.mode
      const [loansRes, repaysRes, interestsRes, liquidationsRes] = await Promise.all([
        sb.from(tableName('margin_loans', mode)).select('*').eq('user_id', userId).order('time', { ascending: false }),
        sb.from(tableName('margin_repays', mode)).select('*').eq('user_id', userId).order('time', { ascending: false }),
        sb.from(tableName('margin_interests', mode)).select('*').eq('user_id', userId).order('time', { ascending: false }),
        sb.from(tableName('margin_liquidations', mode)).select('*').eq('user_id', userId).order('time', { ascending: false }),
      ])
      if (loansRes.error) throw loansRes.error
      if (repaysRes.error) throw repaysRes.error
      if (interestsRes.error) throw interestsRes.error
      if (liquidationsRes.error) throw liquidationsRes.error
      return {
        loans: (loansRes.data ?? []) as MarginLoan[],
        repays: (repaysRes.data ?? []) as MarginRepay[],
        interests: (interestsRes.data ?? []) as MarginInterest[],
        liquidations: (liquidationsRes.data ?? []) as MarginLiquidation[],
      }
    },
    enabled: !!userId,
    staleTime: 60_000,
    refetchInterval: 60_000,
  })
}
