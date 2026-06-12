'use client'

import { useState, useMemo, useCallback, useRef } from 'react'
import { useRouter } from '@/i18n/navigation'
import { useParams, useSearchParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import dynamic from 'next/dynamic'
import {
  useBotDetail,
  useStartInstance,
  useStopInstance,
  useBotSessions,
  useBotStrategiesState,
  useBotPing,
  useContainerLogs,
  type BBGoOrder,
  type BBGoTrade,
  type BBGoBalance,
} from '@/lib/bbgo/queries'
import {
  useSupabaseTrades,
  useSupabaseClosedOrders,
  useSupabasePnLFromProfits,
  useSupabaseLatestPositions,
  useSupabaseLatestPosition,
  useUnrealizedPnL,
  useSupabaseProfits,
  useSupabasePnL,
  useSupabaseOpenOrders,
  useSupabaseBalances,
  useSupabaseFuturesPositions,
  computeFuturesRealtime,
  tradeKey,
  orderKey,
} from '@/lib/bbgo/supabase-queries'
import { useRealtimeTable } from '@/lib/supabase/use-realtime'
import { useUserId } from '@/components/providers/user-id'
import { OrderRow } from '@/components/user/OrderRow'
import { useMarketData } from '@/lib/bbgo/useWebSocket'
import { useKlineData } from '@/hooks/useKlineData'
import { useTradingMode } from '@/components/providers/trading-mode'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { ErrorBoundary } from '@/components/error-boundary'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import type { TradeMarker, OrderLevel, GridLine } from '@/components/chart/CandlestickChart'
import { BotChartPanel } from '@/components/chart/BotChartPanel'
import { TradeRow } from '@/components/user/TradeRow'
import { PositionCard } from '@/components/user/PositionCard'
import { LeverageEditor } from '@/components/user/LeverageEditor'
import { useStrategyRegistry } from '@/components/providers/strategy-registry'
import { getStrategySchema } from '@/lib/bbgo/strategies'
import { extractGridLines, extractStrategyDetails } from '@/lib/bbgo/strategy-state'
import { buildTradeMarkers, buildOrderLevels } from '@/lib/bbgo/trade-markers'
import { computeSMA, computeEMA, computeBollingerBands, DEFAULT_INDICATORS, type IndicatorConfig } from '@/lib/bbgo/indicators'

import {
  ArrowLeft,
  Play,
  Square,
  TrendingUp,
  TrendingDown,
  DollarSign,
  BarChart3,
  Bot,
  Activity,
  AlertCircle,
  WifiOff,
  Target,
  Crosshair,
} from 'lucide-react'

const DepthChart = dynamic(
  () => import('@/components/chart/DepthChart').then((m) => ({ default: m.DepthChart })),
  { ssr: false, loading: () => <div className="h-[300px] animate-pulse rounded-lg bg-muted" /> }
)

function pnlColor(v: number) {
  return v > 0 ? 'text-trade-up' : v < 0 ? 'text-trade-down' : ''
}

function pnlSign(v: number) {
  return v >= 0 ? '+' : ''
}

export default function BotDetailPage() {
  const t = useTranslations('Bots')
  const router = useRouter()
  const params = useParams<{ id: string }>()
  const searchParams = useSearchParams()
  const userId = useUserId()
  const botId = decodeURIComponent(params.id)
  const rawMode = searchParams.get('mode')
  const { mode: globalMode } = useTradingMode()

  const { data: bot, isLoading: botLoading, isError: botError } = useBotDetail(userId, botId)
  const registry = useStrategyRegistry()
  const strategyRequiresFutures = bot?.strategy ? getStrategySchema(bot.strategy, registry)?.requiresFutures === true : false

  const mode: 'live' | 'paper' = rawMode === 'paper' ? 'paper'
    : rawMode === 'live' ? 'live'
    : globalMode

  const isRunning = bot?.container_status === 'running'
  const exchange = bot?.exchange ?? ''
  const symbol = bot?.symbol || (bot?.config?.symbol as string) || ''

  const [selectedSession, setSelectedSession] = useState('')
  const [klineInterval, setKlineInterval] = useState('1h')
  const [indicators, setIndicators] = useState<IndicatorConfig[]>(DEFAULT_INDICATORS)

  const toggleIndicator = useCallback((id: string) => {
    setIndicators((prev) =>
      prev.map((ic) => (ic.id === id ? { ...ic, enabled: !ic.enabled } : ic))
    )
  }, [])
  const [depthData, setDepthData] = useState<{ bids: Array<{ price: number; volume: number }>; asks: Array<{ price: number; volume: number }> }>({ bids: [], asks: [] })
  const [wsBalances, setWsBalances] = useState<Record<string, BBGoBalance> | null>(null)
  const [wsOpenOrders, setWsOpenOrders] = useState<BBGoOrder[] | null>(null)

  const { data: sessionsData } = useBotSessions(userId, mode, isRunning, botId)
  const sessions = sessionsData?.sessions ?? []
  const firstSession = sessions[0]?.name ?? ''
  const activeSession = selectedSession || firstSession

  // Open orders: Supabase-native (works when container is down)
  const { data: supabaseOpenOrders } = useSupabaseOpenOrders(userId, { symbol: symbol || undefined, mode, strategyInstanceId: botId })
  const openOrdersData = { orders: supabaseOpenOrders ?? [] }

  const { data: closedOrdersData } = useSupabaseClosedOrders(userId, { symbol: symbol || undefined, mode, strategyInstanceId: botId })
  const { data: tradesData } = useSupabaseTrades(userId, { symbol: symbol || undefined, mode, strategyInstanceId: botId })

  // Balances: Supabase-native (works when container is down)
  const { data: supabaseBalances } = useSupabaseBalances(userId, { mode })
  const balancesData = { balances: supabaseBalances ?? {} }
  const { data: strategyStatesData } = useBotStrategiesState(userId, mode, isRunning, botId)
  const { data: pingData } = useBotPing(userId, mode, isRunning, botId)
  const { data: logsData } = useContainerLogs(userId, '100', mode, isRunning)

  // Supabase Realtime: invalidate queries on INSERT instead of polling
  const rtOpts = useMemo(() => ({ mode, enabled: isRunning }), [mode, isRunning])
  useRealtimeTable('trades', userId, [['supabase-trades', userId]], rtOpts)
  useRealtimeTable('orders', userId, [['supabase-closed-orders', userId], ['supabase-open-orders', userId]], rtOpts)
  useRealtimeTable('positions', userId, [['supabase-positions', userId], ['supabase-latest-position', userId]], rtOpts)
  useRealtimeTable('profits', userId, [['supabase-profits', userId], ['supabase-pnl-profits', userId]], rtOpts)
  // Balance realtime: invalidate on balance changes (for paper_balances table)
  useRealtimeTable('balances', userId, [['supabase-balances', userId]], rtOpts)

  // Primary PnL from profits table (bbgo average-cost method)
  const { data: pnlAgg } = useSupabasePnLFromProfits(userId, { symbol: symbol || undefined, strategyInstanceId: botId, mode })
  // Fallback PnL from trades (FIFO) when profits table is empty
  const { data: pnlFallback } = useSupabasePnL(userId, { symbol: symbol || undefined, strategyInstanceId: botId, mode })

  const activeExchange = sessions.find((s) => s.name === activeSession)?.exchange ?? exchange
  const activeExchangeRef = useRef(activeExchange)
  activeExchangeRef.current = activeExchange
  const exchangeRef = useRef(exchange)
  exchangeRef.current = exchange

  const { candles, isLoading: klinesLoading, loadEarlierKlines } = useKlineData({
    userId,
    exchange: activeExchange || exchange,
    symbol,
    interval: klineInterval,
    mode,
    enabled: !!exchange && !!symbol,
  })

  const currentPrice = candles.length > 0 ? candles[candles.length - 1]?.close : undefined

  const { data: latestPositions } = useSupabaseLatestPositions(userId, { symbol: symbol || undefined, strategyInstanceId: botId, mode })
  const { data: latestPosition } = useSupabaseLatestPosition(userId, { symbol: symbol || undefined, strategyInstanceId: botId, mode })
  const { data: unrealized } = useUnrealizedPnL(userId, currentPrice, { symbol: symbol || undefined, strategyInstanceId: botId, mode })
  const { data: profitRows } = useSupabaseProfits(userId, { symbol: symbol || undefined, strategyInstanceId: botId, mode, limit: 200 })
  const { data: futuresPositions } = useSupabaseFuturesPositions(userId, { mode, symbol: symbol || undefined, strategyInstanceId: botId })

  const openFuturesPositions = (futuresPositions ?? []).filter(
    (p) => Math.abs(parseFloat(p.position_amount)) > 0
  )
  const futuresRisk = openFuturesPositions[0]
  const hasFuturesOrder = (supabaseOpenOrders ?? []).some((o) => o.isFutures)
    || (closedOrdersData ?? []).some((o) => o.isFutures)
  const isFutures = openFuturesPositions.length > 0 || hasFuturesOrder

  // For futures bots, the positions table (used by useUnrealizedPnL) often holds base=0 rows
  // because the paper engine writes both zero and non-zero snapshots at the same timestamp.
  // Compute the top-line unrealized PnL directly from futures_position_risks via the same
  // computeFuturesRealtime helper used by PositionCard, so the summary matches the cards.
  const futuresUnrealized = useMemo(() => {
    if (!isFutures || openFuturesPositions.length === 0) return null
    let sum = 0
    for (const r of openFuturesPositions) {
      sum += computeFuturesRealtime(r, currentPrice).unrealizedPnl
    }
    return Math.round(sum * 1e6) / 1e6
  }, [isFutures, openFuturesPositions, currentPrice])

  // Use profits-based PnL when available, fall back to FIFO
  const hasProfitsData = (pnlAgg?.totalTrades ?? 0) > 0
  const pnl = hasProfitsData ? pnlAgg : null
  const pnlLegacy = !hasProfitsData ? pnlFallback : null

  const strategyMatch = useCallback(
    (strategyInstanceId: string | undefined) => !!bot && strategyInstanceId === bot.id,
    [bot?.id]
  )

  const tradeOrderIds = useMemo(() => {
    const ids = new Set<number>()
    for (const t of tradesData ?? []) {
      if (t.orderID) ids.add(t.orderID)
    }
    return ids
  }, [tradesData])

  const allOpenOrders = wsOpenOrders ?? openOrdersData?.orders ?? []
  const allClosedOrders = closedOrdersData ?? []
  const openOrders = allOpenOrders.filter((o) => strategyMatch(o.strategyInstanceId))
  const closedOrders = allClosedOrders.filter(
    (o) => strategyMatch(o.strategyInstanceId) || tradeOrderIds.has(o.orderID)
  )

  const tradeMarkers: TradeMarker[] = useMemo(
    () => buildTradeMarkers(tradesData ?? null, closedOrders, symbol, isFutures),
    [tradesData, closedOrders, symbol, isFutures]
  )

  const orderLevels: OrderLevel[] = useMemo(
    () => buildOrderLevels(openOrders, symbol),
    [openOrders, symbol]
  )

  const findMatchingStrategy = useCallback((strategies: Record<string, unknown>[]) => {
    if (!strategies.length) return undefined
    if (strategies.length === 1) return strategies[0]
    if (bot?.strategy) {
      const matched = strategies.find((s) => s.strategy === bot.strategy)
      if (matched) return matched
    }
    const botLower = bot?.config?.lowerPrice as number | undefined
    const botUpper = bot?.config?.upperPrice as number | undefined
    if (botLower != null && botUpper != null) {
      const matched = strategies.find((s) => {
        const strategy = s.strategy as string
        const inner = s[strategy] as Record<string, unknown> | undefined
        return inner?.lowerPrice === botLower && inner?.upperPrice === botUpper
      })
      if (matched) return matched
    }
    return strategies.find((s) => {
      const strategy = s.strategy as string
      const inner = s[strategy] as Record<string, unknown> | undefined
      return inner?.symbol === symbol || (!symbol && inner?.symbol)
    })
  }, [bot?.strategy, bot?.config?.lowerPrice, bot?.config?.upperPrice, symbol])

  const gridLines: GridLine[] = useMemo(() => {
    if (!strategyStatesData?.strategies) return []
    const matching = findMatchingStrategy(strategyStatesData.strategies as Record<string, unknown>[])
    if (!matching) return []
    const lines = extractGridLines(matching as Record<string, unknown>, currentPrice)
    const strategyKey = matching['strategy'] as string | undefined
    const posState = strategyKey ? (matching[strategyKey] as Record<string, unknown>)?.Position as Record<string, unknown> | undefined : undefined
    if (posState?.averageCost) {
      const ac = typeof posState.averageCost === 'number' ? posState.averageCost : parseFloat(String(posState.averageCost))
      const base = typeof posState.base === 'number' ? posState.base : parseFloat(String(posState.base ?? '0'))
      if (ac > 0 && base > 0) {
        lines.push({ price: ac, label: t('avgCost', { price: ac.toLocaleString() }), color: 'rgba(251, 146, 60, 0.7)' })
      }
    }
    return lines
  }, [strategyStatesData, currentPrice, findMatchingStrategy, t])

  const indicatorLines = useMemo(() => {
    if (candles.length === 0) return []
    const closes = candles.map((c) => ({ time: c.time, close: c.close }))
    const lines: Array<{
      id: string
      name: string
      color: string
      lineWidth?: number
      lineStyle?: number
      data: Array<{ time: import('lightweight-charts').Time; value: number }>
    }> = []

    for (const ic of indicators) {
      if (!ic.enabled) continue
      if (ic.type === 'sma') {
        const data = computeSMA(closes, ic.period)
        if (data.length > 0) lines.push({ id: ic.id, name: ic.name, color: ic.color, data })
      } else if (ic.type === 'ema') {
        const data = computeEMA(closes, ic.period)
        if (data.length > 0) lines.push({ id: ic.id, name: ic.name, color: ic.color, data })
      } else if (ic.type === 'bollinger') {
        const { upper, middle, lower } = computeBollingerBands(closes, ic.period, 2)
        if (middle.length > 0) {
          lines.push({ id: `${ic.id}-upper`, name: `${ic.name} Upper`, color: ic.color, lineStyle: 2, data: upper })
          lines.push({ id: `${ic.id}-mid`, name: ic.name, color: ic.color, data: middle })
          lines.push({ id: `${ic.id}-lower`, name: `${ic.name} Lower`, color: ic.color, lineStyle: 2, data: lower })
        }
      }
    }
    return lines
  }, [candles, indicators])

  const pnlLine = useMemo(() => {
    const curve = pnl?.pnlCurve ?? pnlLegacy?.pnlCurve
    if (!curve || curve.length < 2) return null
    return {
      id: 'pnl-curve', name: t('pnl.realized'), color: '#a855f7', lineWidth: 2,
      priceScaleId: 'pnl', scaleMargins: { top: 0.75, bottom: 0 },
      data: curve.map((p) => ({ time: p.time as import('lightweight-charts').Time, value: p.value })),
    }
  }, [pnl?.pnlCurve, pnlLegacy?.pnlCurve, t])

  const strategyStats = useMemo(() => {
    if (!strategyStatesData?.strategies) return null
    const matching = findMatchingStrategy(strategyStatesData.strategies as Record<string, unknown>[])
    if (!matching) return null
    return extractStrategyDetails(matching as Record<string, unknown>)
  }, [strategyStatesData, findMatchingStrategy])

  interface DepthMessage {
    type: string
    data: {
      channel?: string
      depth?: { bids: Array<{ price: string; volume: string }>; asks: Array<{ price: string; volume: string }> }
      balances?: Array<{ currency: string; available: string; locked: string }>
      orders?: Array<{ id: string; symbol: string; side: string; orderType?: string; price: string; quantity: string; executedQuantity: string; status: string }>
    }
  }

  const handleWSMessage = useCallback((msg: DepthMessage) => {
    if (msg.type === 'market' && msg.data.depth) {
      setDepthData({
        bids: msg.data.depth.bids.slice(0, 20).map((b) => ({ price: parseFloat(b.price), volume: parseFloat(b.volume) })),
        asks: msg.data.depth.asks.slice(0, 20).map((a) => ({ price: parseFloat(a.price), volume: parseFloat(a.volume) })),
      })
    }
    // Manager WebSocket userData: balance and order updates from bbgo container
    if (msg.type === 'userData') {
      if (msg.data.channel === 'BALANCE' && msg.data.balances) {
        const record: Record<string, BBGoBalance> = {}
        for (const b of msg.data.balances) {
          record[b.currency] = { currency: b.currency, available: b.available, locked: b.locked }
        }
        setWsBalances(record)
      }
      if (msg.data.channel === 'ORDER' && msg.data.orders) {
        setWsOpenOrders(msg.data.orders.map((o) => ({
          gid: 0,
          orderID: parseInt(o.id) || 0,
          exchange: activeExchangeRef.current || exchangeRef.current,
          symbol: o.symbol,
          side: o.side as 'BUY' | 'SELL',
          orderType: (o.orderType as BBGoOrder['orderType']) || 'LIMIT',
          price: o.price,
          quantity: o.quantity,
          executedQuantity: o.executedQuantity,
          status: o.status,
          clientOrderID: '',
          isWorking: o.status === 'NEW',
        })))
      }
    }
  }, [])

  const { connected: wsConnected } = useMarketData({
    userId,
    mode,
    enabled: isRunning,
    onMessage: handleWSMessage,
  })

  const startInstance = useStartInstance()
  const stopInstance = useStopInstance()

  const trades = useMemo(() => tradesData ?? [], [tradesData])
  // Derived PnL values for display
  const netProfit = pnl?.totalNetProfit ?? pnlLegacy?.totalRealizedPnl ?? 0
  const unrealizedPnl = futuresUnrealized ?? unrealized?.unrealizedPnl ?? pnlLegacy?.totalUnrealizedPnl ?? 0
  const totalFees = pnl?.totalFees ?? pnlLegacy?.totalFees ?? 0
  const winRate = pnl?.winRate ?? pnlLegacy?.winRate ?? 0
  const profitFactor = pnl?.profitFactor ?? pnlLegacy?.profitFactor ?? 0
  const totalTrades = pnl?.totalTrades ?? pnlLegacy?.totalTrades ?? 0
  const winningTrades = pnl?.profitCount ?? pnlLegacy?.winningTrades ?? 0
  const losingTrades = pnl?.lossCount ?? pnlLegacy?.losingTrades ?? 0

  if (botLoading || !userId) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => <Skeleton key={i} className="h-20 rounded-xl" />)}
        </div>
      </div>
    )
  }

  if (botError || !bot) {
    return (
      <Card className="rounded-xl border-destructive/50">
        <CardContent className="flex flex-col items-center py-12">
          <AlertCircle className="h-8 w-8 text-destructive mb-3" />
          <p className="text-sm text-destructive">{t('errorLoading')}</p>
        </CardContent>
      </Card>
    )
  }

  const status = bot.container_status
  const botReachable = isRunning && pingData?.status === 'ok'
  const balances = wsBalances ?? balancesData?.balances ?? {}
  const liveStrategies = strategyStatesData?.strategies ?? []
  const nonZeroBalances = Object.entries(balances).filter(
    ([, b]) => parseFloat(b.available) > 0 || parseFloat(b.locked) > 0
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div className="space-y-1 min-w-0">
          <button
            onClick={() => router.back()}
            className="inline-flex items-center gap-1 text-[13px] text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            {t('backToBots')}
          </button>
          <h1 className="text-2xl font-semibold tracking-tight truncate">{bot.strategy}</h1>
          <p className="text-sm text-muted-foreground">
            {exchange}{symbol ? ` · ${symbol}` : ''} · {bot.strategy} · {t(`mode.${mode}`)}
          </p>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {isRunning && (
            <Badge
              variant="outline"
              className={cn(
                'gap-1.5 rounded-full text-[11px]',
                wsConnected
                  ? 'border-trade-up/30 text-trade-up'
                  : 'border-yellow-500/30 text-yellow-600'
              )}
            >
              {wsConnected ? (
                <><span className="h-1.5 w-1.5 rounded-full bg-trade-up animate-pulse" />{t('live.connected')}</>
              ) : (
                <><span className="h-1.5 w-1.5 rounded-full bg-yellow-500 animate-pulse" />{t('live.connecting')}</>
              )}
            </Badge>
          )}

          <Badge
            variant="outline"
            className={cn(
              'rounded-full text-[11px] font-medium',
              status === 'running' && 'border-trade-up/30 text-trade-up',
              status === 'stopped' && 'border-border text-muted-foreground',
              status === 'error' && 'border-trade-down/30 text-trade-down'
            )}
          >
            {t(`status.${status}`)}
          </Badge>

          {status === 'running' ? (
            <Button
              variant="outline"
              size="sm"
              onClick={() => stopInstance.mutate({ userId, instanceId: bot.id }, { onError: (err) => toast.error(err.message) })}
              disabled={stopInstance.isPending}
              className="rounded-full"
            >
              <Square className="mr-1.5 h-3.5 w-3.5" />
              {t('stop')}
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={() => startInstance.mutate({ userId, instanceId: bot.id }, { onError: (err) => toast.error(err.message) })}
              disabled={startInstance.isPending || status === 'starting'}
              className="rounded-full"
            >
              <Play className="mr-1.5 h-3.5 w-3.5" />
              {t('start')}
            </Button>
          )}
        </div>
      </div>

      {!isRunning && (
        <div
          className={cn(
            'flex items-center gap-3 rounded-xl px-5 py-3.5 text-sm',
            status === 'error'
              ? 'bg-trade-down/5 text-trade-down border border-trade-down/20'
              : 'bg-muted text-muted-foreground'
          )}
        >
          {status === 'error' ? <AlertCircle className="h-4 w-4 shrink-0" /> : <WifiOff className="h-4 w-4 shrink-0" />}
          {status === 'error' ? t('errorBanner') : t('stoppedBanner')}
        </div>
      )}

      {sessions.length > 0 && (
        <div className="flex gap-2">
          {sessions.map((s) => (
            <Button
              key={s.name}
              variant={activeSession === s.name ? 'default' : 'outline'}
              size="sm"
              onClick={() => setSelectedSession(s.name)}
              className="rounded-full text-xs"
            >
              {s.exchange || s.name}
            </Button>
          ))}
        </div>
      )}

      {/* PnL Summary Cards — from profits table (bbgo average-cost) */}
      {(totalTrades > 0 || unrealizedPnl !== 0) && (
        <div className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.netProfit')}</p>
                <div className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full',
                  netProfit >= 0 ? 'bg-trade-up/10' : 'bg-trade-down/10'
                )}>
                  {netProfit >= 0
                    ? <TrendingUp className="h-3.5 w-3.5 text-trade-up" />
                    : <TrendingDown className="h-3.5 w-3.5 text-trade-down" />}
                </div>
              </div>
              <p className={cn('mt-2 text-lg font-semibold font-mono', pnlColor(netProfit))}>
                {pnlSign(netProfit)}{netProfit.toFixed(4)}
              </p>
            </CardContent>
          </Card>

          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.unrealized')}</p>
                <div className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full',
                  unrealizedPnl >= 0 ? 'bg-trade-up/10' : 'bg-trade-down/10'
                )}>
                  <Crosshair className="h-3.5 w-3.5 text-muted-foreground" />
                </div>
              </div>
              <p className={cn('mt-2 text-lg font-semibold font-mono', pnlColor(unrealizedPnl))}>
                {pnlSign(unrealizedPnl)}{unrealizedPnl.toFixed(4)}
              </p>
            </CardContent>
          </Card>

          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.totalFees')}</p>
                <DollarSign className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-lg font-semibold font-mono text-muted-foreground">
                -{totalFees.toFixed(4)}
              </p>
            </CardContent>
          </Card>

          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.winRate')}</p>
                <BarChart3 className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-lg font-semibold font-mono">
                {winRate.toFixed(1)}%
                <span className="ml-2 text-xs text-muted-foreground font-normal">
                  ({t('pnl.winLossFormat', { wins: winningTrades, losses: losingTrades })})
                </span>
              </p>
            </CardContent>
          </Card>

          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.profitFactor')}</p>
                <Target className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className={cn('mt-2 text-lg font-semibold font-mono', pnlColor(profitFactor - 1))}>
                {profitFactor === Infinity ? '∞' : profitFactor.toFixed(2)}
              </p>
            </CardContent>
          </Card>

          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.totalTrades')}</p>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-lg font-semibold font-mono">{totalTrades}</p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Position — unified cards for spot and futures */}
      {strategyRequiresFutures && bot && (
        <LeverageEditor
          instanceId={botId ?? ''}
          strategy={bot.strategy}
          symbol={symbol ?? ''}
          currentLeverage={futuresRisk ? parseFloat(futuresRisk.leverage) : undefined}
          requiresFutures
        />
      )}
      <PositionCard
        spotPositions={latestPositions ?? []}
        futuresRisks={futuresPositions ?? []}
        spotUnrealized={unrealized}
        currentPrice={currentPrice}
        isFutures={isFutures}
      />

      <Tabs defaultValue="chart" className="space-y-4">
        <TabsList className="bg-muted/50 p-1 rounded-lg w-full overflow-x-auto">
          <TabsTrigger value="chart" className="rounded-md text-xs">{t('chart')}</TabsTrigger>
          <TabsTrigger value="depth" className="rounded-md text-xs">{t('depth')}</TabsTrigger>
          <TabsTrigger value="balances" className="rounded-md text-xs">{t('balances')}</TabsTrigger>
          <TabsTrigger value="open-orders" className="rounded-md text-xs">{t('openOrders')} ({openOrders.length})</TabsTrigger>
          <TabsTrigger value="closed-orders" className="rounded-md text-xs">{t('closedOrders')} ({closedOrders.length})</TabsTrigger>
          <TabsTrigger value="trades" className="rounded-md text-xs">{t('recentTrades')}</TabsTrigger>
          {(profitRows?.length ?? 0) > 0 && (
            <TabsTrigger value="close-history" className="rounded-md text-xs">{t('pnl.closeHistory')}</TabsTrigger>
          )}
          <TabsTrigger value="strategies" className="rounded-md text-xs">{t('strategies')}</TabsTrigger>
          {isRunning && <TabsTrigger value="logs" className="rounded-md text-xs">{t('containerLogs')}</TabsTrigger>}
        </TabsList>

        <TabsContent value="chart">
          <BotChartPanel
            symbol={symbol}
            exchange={activeExchange}
            botReachable={botReachable}
            candles={candles}
            tradeMarkers={tradeMarkers}
            orderLevels={orderLevels}
            gridLines={gridLines}
            indicatorLines={indicatorLines}
            pnlLine={pnlLine}
            klinesLoading={klinesLoading}
            loadEarlierKlines={loadEarlierKlines}
            strategyStats={strategyStats}
            currentPrice={currentPrice}
            unrealizedPnlFromReport={unrealizedPnl}
            noSymbolText={t('noSymbolForChart')}
            startToSeeDataText={t('startToSeeData')}
            klineInterval={klineInterval}
            onIntervalChange={setKlineInterval}
            indicatorConfigs={indicators}
            onToggleIndicator={toggleIndicator}
            supabasePosition={latestPosition ? {
              base: latestPosition.base,
              averageCost: latestPosition.averageCost,
              quote: latestPosition.quote,
              symbol: latestPosition.symbol,
              isClosed: latestPosition.isClosed,
              ...(futuresRisk ? {
                leverage: parseFloat(futuresRisk.leverage),
                liquidationPrice: parseFloat(futuresRisk.liquidation_price) || undefined,
                direction: parseFloat(futuresRisk.position_amount) > 0 ? 'long' as const : parseFloat(futuresRisk.position_amount) < 0 ? 'short' as const : 'flat' as const,
              } : {}),
            } : undefined}
            unrealizedPnlPct={unrealized?.unrealizedPnlPct}
          />
        </TabsContent>

        <TabsContent value="depth">
          <ErrorBoundary>
          <Card className="rounded-xl">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium">{t('orderBook')}</CardTitle>
            </CardHeader>
            <CardContent>
              <DepthChart bids={depthData.bids} asks={depthData.asks} height={350} />
            </CardContent>
          </Card>
          </ErrorBoundary>
        </TabsContent>

        <TabsContent value="balances">
          <Card className="rounded-xl">
            {nonZeroBalances.length > 0 ? (
              <div className="divide-y">
                {nonZeroBalances.map(([currency, b]: [string, BBGoBalance]) => (
                  <div key={currency} className="flex items-center justify-between px-6 py-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-xs font-semibold">{currency.slice(0, 2)}</div>
                      <span className="text-sm font-medium">{currency}</span>
                    </div>
                    <div className="text-right">
                      <span className="text-sm font-mono">{b.available}</span>
                      {parseFloat(b.locked) > 0 && (
                        <span className="ml-2 text-xs text-muted-foreground">({t('locked', { amount: b.locked })})</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {isRunning ? t('noBalances') : t('startToSeeData')}
              </CardContent>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="open-orders">
          <Card className="rounded-xl">
            {openOrders.length > 0 ? (
              <div className="divide-y">
                {openOrders.map((order: BBGoOrder) => (
                  <OrderRow key={orderKey(order)} order={order} />
                ))}
              </div>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {isRunning ? t('noOpenOrders') : t('startToSeeData')}
              </CardContent>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="closed-orders">
          <Card className="rounded-xl">
            {closedOrders.length > 0 ? (
              <ScrollArea className="max-h-[400px]">
                <div className="divide-y">
                  {closedOrders.map((order: BBGoOrder) => (
                    <OrderRow key={orderKey(order)} order={order} showStatus showTime />
                  ))}
                </div>
              </ScrollArea>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {isRunning ? t('noClosedOrders') : t('startToSeeData')}
              </CardContent>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="trades">
          <Card className="rounded-xl">
            {trades.length > 0 ? (
              <ScrollArea className="max-h-[400px]">
                <div className="divide-y">
                  {trades.map((trade: BBGoTrade) => (
                    <TradeRow
                      key={tradeKey(trade)}
                      trade={trade}
                      netPosition={trade.netPosition ?? 0}
                      isFutures={isFutures}
                    />
                  ))}
                </div>
              </ScrollArea>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {isRunning ? t('noTrades') : t('startToSeeData')}
              </CardContent>
            )}
          </Card>
        </TabsContent>

        {/* Close History — from profits table (bbgo pre-computed) */}
        <TabsContent value="close-history">
          <Card className="rounded-xl">
            {(profitRows?.length ?? 0) > 0 ? (
              <ScrollArea className="max-h-[400px]">
                <div className="divide-y">
                  {profitRows!.map((row) => {
                    const profit = parseFloat(row.profit ?? '0')
                    const netProfitVal = parseFloat(row.net_profit ?? '0')
                    const profitMargin = parseFloat(row.profit_margin ?? '0')
                    return (
                      <div key={row.id} className="flex items-center justify-between px-6 py-3">
                        <div className="flex items-center gap-3 min-w-0">
                          <div className={cn(
                            'flex h-6 w-6 items-center justify-center rounded text-xs font-bold',
                            netProfitVal >= 0 ? 'bg-trade-up/10 text-trade-up' : 'bg-trade-down/10 text-trade-down'
                          )}>
                            {netProfitVal >= 0 ? 'W' : 'L'}
                          </div>
                          <div className="flex flex-col gap-0.5 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium">{row.symbol}</span>
                              <span className="text-xs text-muted-foreground">{row.strategy}</span>
                            </div>
                            <span className="text-xs text-muted-foreground tabular-nums">{row.traded_at ? new Date(row.traded_at).toLocaleString() : ''}</span>
                          </div>
                        </div>
                        <div className="text-right space-y-0.5 shrink-0">
                          <p className={cn('text-sm font-mono font-medium', pnlColor(netProfitVal))}>
                            {pnlSign(netProfitVal)}{netProfitVal.toFixed(4)}
                          </p>
                          <div className="flex items-center justify-end gap-3 text-xs text-muted-foreground">
                            <span>{t('pnl.profitMargin')}: {profitMargin.toFixed(2)}%</span>
                            <span>{t('pnl.realized')}: {pnlSign(profit)}{profit.toFixed(4)}</span>
                          </div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </ScrollArea>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {t('pnl.noData')}
              </CardContent>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="strategies">
          <Card className="rounded-xl">
            {liveStrategies.length > 0 ? (
              <div className="divide-y">
                {liveStrategies.map((ls, idx) => (
                  <div key={idx} className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                          <Bot className="h-4 w-4 text-muted-foreground" />
                        </div>
                        <div>
                          <p className="text-sm font-medium">{ls.strategy}</p>
                          <p className="text-xs text-muted-foreground">{exchange} · {t(`mode.${mode}`)}</p>
                        </div>
                      </div>
                      <Badge
                        variant="default"
                        className="rounded-full text-[11px] bg-trade-up text-white hover:bg-trade-up/90"
                      >
                        {t('strategyStatus.running')}
                      </Badge>
                    </div>
                    {Object.keys(ls).length > 1 && (
                      <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 pl-11">
                        {Object.entries(ls)
                          .filter(([k, v]) => k !== 'strategy' && v != null && typeof v !== 'object' && typeof v !== 'function')
                          .slice(0, 10)
                          .map(([key, val]) => (
                            <span key={key} className="text-xs text-muted-foreground">
                              {key}: <span className="text-foreground font-mono">{String(val)}</span>
                            </span>
                          ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">{t('noStrategiesTab')}</CardContent>
            )}
          </Card>
        </TabsContent>

        {isRunning && (
          <TabsContent value="logs">
            <Card className="rounded-xl">
              {logsData?.logs ? (
                <pre className="whitespace-pre-wrap text-xs text-muted-foreground max-h-[400px] overflow-y-auto p-5 font-mono leading-relaxed">
                  {logsData.logs.replace(/\x1b\[[0-9;]*m/g, '')}
                </pre>
              ) : (
                <CardContent className="py-8 text-center text-sm text-muted-foreground">{t('loadingLogs')}</CardContent>
              )}
            </Card>
          </TabsContent>
        )}
      </Tabs>
    </div>
  )
}
