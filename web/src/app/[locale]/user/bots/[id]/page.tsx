'use client'

import { useState, useMemo, useCallback } from 'react'
import { useRouter } from '@/i18n/navigation'
import { useParams, useSearchParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import dynamic from 'next/dynamic'
import {
  useBotDetail,
  useStartUser,
  useStopUser,
  useBotSessions,
  useBotOpenOrders,
  useBotClosedOrders,
  useBotTrades,
  useBotTradeMarkers,
  useBotSessionBalances,
  useBotStrategiesState,
  useBotPing,
  useContainerLogs,
  useBotPnL,
  type BBGoOrder,
  type BBGoTrade,
  type BBGoBalance,
} from '@/lib/bbgo/queries'
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
import { extractGridLines, extractStrategyStats } from '@/lib/bbgo/strategy-state'
import { buildTradeMarkers, buildTradeMarkersFromServer, buildOrderLevels } from '@/lib/bbgo/trade-markers'
import { computePositionTags } from '@/lib/bbgo/position-tags'
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
} from 'lucide-react'

const DepthChart = dynamic(
  () => import('@/components/chart/DepthChart').then((m) => ({ default: m.DepthChart })),
  { ssr: false, loading: () => <div className="h-[300px] animate-pulse rounded-lg bg-muted" /> }
)

export default function BotDetailPage() {
  const t = useTranslations('Bots')
  const router = useRouter()
  const params = useParams<{ id: string }>()
  const searchParams = useSearchParams()
  const userId = useUserId()
  const botId = params.id
  const rawMode = searchParams.get('mode')
  const { mode: globalMode } = useTradingMode()

  const { data: bot, isLoading: botLoading, isError: botError } = useBotDetail(userId, botId)

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

  const { data: sessionsData } = useBotSessions(userId, mode, isRunning)
  const sessions = sessionsData?.sessions ?? []
  const firstSession = sessions[0]?.name ?? ''
  const activeSession = selectedSession || firstSession

  const { data: openOrdersData } = useBotOpenOrders(userId, activeSession, mode, isRunning)
  const { data: closedOrdersData } = useBotClosedOrders(userId, undefined, symbol || undefined, mode, isRunning)
  const { data: tradesData } = useBotTrades(userId, undefined, symbol || undefined, mode, isRunning)
  const { data: tradeMarkersData } = useBotTradeMarkers(userId, symbol || '', { exchange: undefined, mode, limit: 200 }, isRunning)
  const { data: balancesData } = useBotSessionBalances(userId, activeSession, mode, isRunning)
  const { data: strategyStatesData } = useBotStrategiesState(userId, mode, isRunning)
  const { data: pingData } = useBotPing(userId, mode, isRunning)
  const { data: logsData } = useContainerLogs(userId, '100', mode, isRunning)
  const { data: pnlData } = useBotPnL(userId, undefined, symbol || undefined, mode, isRunning)

  const activeExchange = sessions.find((s) => s.name === activeSession)?.exchange ?? exchange

  const { candles, isLoading: klinesLoading, loadEarlierKlines } = useKlineData({
    userId,
    exchange: activeExchange || exchange,
    symbol,
    interval: klineInterval,
    mode,
    enabled: !!exchange && !!symbol,
  })

  const tradeMarkers: TradeMarker[] = useMemo(
    () => { const server = buildTradeMarkersFromServer(tradeMarkersData); return server?.length ? server : buildTradeMarkers(tradesData?.trades ?? null, closedOrdersData?.orders ?? null, symbol) },
    [tradeMarkersData, tradesData?.trades, closedOrdersData?.orders, symbol]
  )

  const orderLevels: OrderLevel[] = useMemo(
    () => buildOrderLevels(openOrdersData?.orders ?? null, symbol),
    [openOrdersData?.orders, symbol]
  )

  const currentPrice = candles.length > 0 ? candles[candles.length - 1]?.close : undefined

  const findMatchingStrategy = useCallback((strategies: Record<string, unknown>[]) => {
    if (!strategies.length) return undefined
    // If only one strategy, use it
    if (strategies.length === 1) return strategies[0]
    // Match by bot config (lowerPrice/upperPrice) to disambiguate multiple strategies on same symbol
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
    // Fallback: first strategy matching symbol
    return strategies.find((s) => {
      const strategy = s.strategy as string
      const inner = s[strategy] as Record<string, unknown> | undefined
      return inner?.symbol === symbol || (!symbol && inner?.symbol)
    })
  }, [bot?.config?.lowerPrice, bot?.config?.upperPrice, symbol])

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
    if (!pnlData?.pnlCurve || pnlData.pnlCurve.length < 2) return null
    return {
      id: 'pnl-curve', name: t('pnl.realized'), color: '#a855f7', lineWidth: 2,
      priceScaleId: 'pnl', scaleMargins: { top: 0.75, bottom: 0 },
      data: pnlData.pnlCurve.map((p) => ({ time: p.time as import('lightweight-charts').Time, value: p.value })),
    }
  }, [pnlData?.pnlCurve, t])

  const strategyStats = useMemo(() => {
    if (!strategyStatesData?.strategies) return null
    const matching = findMatchingStrategy(strategyStatesData.strategies as Record<string, unknown>[])
    if (!matching) return null
    const stats = extractStrategyStats(matching as Record<string, unknown>)
    if (!stats) return null

    // Fallback: when bbgo strategy state reports zero position but trades show otherwise,
    // use the trade-computed position from PnL data
    if (stats.base === 0 && pnlData?.symbols) {
      const symPnl = pnlData.symbols.find((s) => s.symbol === symbol)
      if (symPnl && symPnl.openPosition > 0) {
        return {
          ...stats,
          base: symPnl.openPosition,
          quote: symPnl.openPositionCost,
          averageCost: symPnl.openPositionCost / symPnl.openPosition,
        }
      }
    }
    return stats
  }, [strategyStatesData, findMatchingStrategy, pnlData?.symbols, symbol])

  interface DepthMessage {
    type: string
    data: {
      channel?: string
      depth?: { bids: Array<{ price: string; volume: string }>; asks: Array<{ price: string; volume: string }> }
    }
  }

  const handleWSMessage = useCallback((msg: DepthMessage) => {
    if (msg.type !== 'market' || !msg.data.depth) return
    setDepthData({
      bids: msg.data.depth.bids.slice(0, 20).map((b) => ({ price: parseFloat(b.price), volume: parseFloat(b.volume) })),
      asks: msg.data.depth.asks.slice(0, 20).map((a) => ({ price: parseFloat(a.price), volume: parseFloat(a.volume) })),
    })
  }, [])

  const { connected: wsConnected } = useMarketData({
    userId,
    mode,
    enabled: isRunning,
    onMessage: handleWSMessage,
  })

  const startUser = useStartUser()
  const stopUser = useStopUser()

  const trades = useMemo(() => tradesData?.trades ?? [], [tradesData?.trades])
  const tradePositionTags = useMemo(() => {
    if (trades.some((t) => t.positionAction)) return null
    return computePositionTags(trades)
  }, [trades])

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
  const openOrders = openOrdersData?.orders ?? []
  const closedOrders = closedOrdersData?.orders ?? []
  const balances = balancesData?.balances ?? {}
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
              onClick={() => stopUser.mutate({ userId, mode }, { onError: (err) => toast.error(err.message) })}
              disabled={stopUser.isPending}
              className="rounded-full"
            >
              <Square className="mr-1.5 h-3.5 w-3.5" />
              {t('stop')}
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={() => startUser.mutate({ userId, mode }, { onError: (err) => toast.error(err.message) })}
              disabled={startUser.isPending || status === 'starting'}
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

      {botReachable && pnlData && pnlData.totalTrades > 0 && (
        <div className="grid gap-4 md:grid-cols-5">
          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.realized')}</p>
                <div className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full',
                  pnlData.totalRealizedPnl >= 0 ? 'bg-trade-up' : 'bg-trade-down'
                )}>
                  {pnlData.totalRealizedPnl >= 0
                    ? <TrendingUp className="h-3.5 w-3.5 text-trade-up" />
                    : <TrendingDown className="h-3.5 w-3.5 text-trade-down" />}
                </div>
              </div>
              <p className={cn(
                'mt-2 text-xl font-semibold font-mono',
                pnlData.totalRealizedPnl > 0 ? 'text-trade-up' : pnlData.totalRealizedPnl < 0 ? 'text-trade-down' : ''
              )}>
                {pnlData.totalRealizedPnl >= 0 ? '+' : ''}{pnlData.totalRealizedPnl.toFixed(4)} USDT
              </p>
            </CardContent>
          </Card>

          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.unrealized')}</p>
                <div className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full',
                  pnlData.totalUnrealizedPnl >= 0 ? 'bg-trade-up' : 'bg-trade-down'
                )}>
                  {pnlData.totalUnrealizedPnl >= 0
                    ? <TrendingUp className="h-3.5 w-3.5 text-trade-up" />
                    : <TrendingDown className="h-3.5 w-3.5 text-trade-down" />}
                </div>
              </div>
              <p className={cn(
                'mt-2 text-xl font-semibold font-mono',
                pnlData.totalUnrealizedPnl > 0 ? 'text-trade-up' : pnlData.totalUnrealizedPnl < 0 ? 'text-trade-down' : ''
              )}>
                {pnlData.totalUnrealizedPnl >= 0 ? '+' : ''}{pnlData.totalUnrealizedPnl.toFixed(4)} USDT
              </p>
            </CardContent>
          </Card>
          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.totalFees')}</p>
                <DollarSign className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-xl font-semibold font-mono text-muted-foreground">
                -{pnlData.totalFees.toFixed(4)} USDT
              </p>
            </CardContent>
          </Card>
          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.winRate')}</p>
                <BarChart3 className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-xl font-semibold font-mono">
                {pnlData.winRate.toFixed(1)}%
                <span className="ml-2 text-xs text-muted-foreground font-normal">
                  ({t('pnl.winLossFormat', { wins: pnlData.winningTrades, losses: pnlData.losingTrades })})
                </span>
              </p>
            </CardContent>
          </Card>
          <Card className="rounded-xl">
            <CardContent className="p-5">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-medium text-muted-foreground">{t('pnl.totalTrades')}</p>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-xl font-semibold font-mono">{pnlData.totalTrades}</p>
            </CardContent>
          </Card>
        </div>
      )}

      {botReachable && pnlData && pnlData.symbols?.length > 0 && (
        <Card className="rounded-xl">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">{t('pnl.bySymbol')}</CardTitle>
          </CardHeader>
          <div className="divide-y">
            {pnlData.symbols.map((s) => (
              <div key={s.symbol} className="flex items-center justify-between px-6 py-3 text-sm">
                <div className="flex items-center gap-3 min-w-[140px]">
                  <span className="font-medium">{s.symbol}</span>
                  <span className="text-xs text-muted-foreground">{t('pnl.tradeCount', { count: s.tradeCount })}</span>
                </div>
                <div className="flex items-center gap-6">
                  {s.openPosition > 0 && (
                    <span className="text-xs text-muted-foreground">
                      {t('pnl.openPositionNoPrice', { amount: s.openPosition.toFixed(6) })}
                    </span>
                  )}
                  {s.unrealizedPnl !== 0 && (
                    <span className={cn(
                      'text-xs font-mono',
                      s.unrealizedPnl > 0 ? 'text-trade-up' : 'text-trade-down'
                    )}>
                      {t('pnl.unrealized')}: {s.unrealizedPnl >= 0 ? '+' : ''}{s.unrealizedPnl.toFixed(4)}
                    </span>
                  )}
                  <span className={cn(
                    'font-medium w-32 text-right font-mono',
                    s.realizedPnl > 0 ? 'text-trade-up' : s.realizedPnl < 0 ? 'text-trade-down' : ''
                  )}>
                    {s.realizedPnl >= 0 ? '+' : ''}{s.realizedPnl.toFixed(4)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      <Tabs defaultValue="chart" className="space-y-4">
        <TabsList className="bg-muted/50 p-1 rounded-lg w-full overflow-x-auto">
          <TabsTrigger value="chart" className="rounded-md text-xs">{t('chart')}</TabsTrigger>
          <TabsTrigger value="depth" className="rounded-md text-xs">{t('depth')}</TabsTrigger>
          <TabsTrigger value="balances" className="rounded-md text-xs">{t('balances')}</TabsTrigger>
          <TabsTrigger value="open-orders" className="rounded-md text-xs">{t('openOrders')} ({openOrders.length})</TabsTrigger>
          <TabsTrigger value="closed-orders" className="rounded-md text-xs">{t('closedOrders')} ({closedOrders.length})</TabsTrigger>
          <TabsTrigger value="trades" className="rounded-md text-xs">{t('recentTrades')}</TabsTrigger>
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
            noSymbolText={t('noSymbolForChart')}
            startToSeeDataText={t('startToSeeData')}
            klineInterval={klineInterval}
            onIntervalChange={setKlineInterval}
            indicatorConfigs={indicators}
            onToggleIndicator={toggleIndicator}
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
                  <OrderRow key={order.orderID} order={order} />
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
                    <OrderRow key={order.orderID} order={order} showStatus showTime />
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
                  {trades.map((trade: BBGoTrade) => {
                    const isBuy = trade.side === 'BUY'
                    const serverTag = trade.positionAction
                    const localTag = tradePositionTags?.[trades.indexOf(trade)]
                    const tag = serverTag ?? localTag?.tag ?? null
                    const netPos = trade.netPosition ?? localTag?.netPos ?? 0
                    return (
                      <div key={trade.id} className={cn(
                        'flex items-center justify-between px-6 py-3 border-l-2',
                        tag === 'open' ? 'border-l-blue-400' : tag === 'close' ? 'border-l-orange-400' : isBuy ? 'border-l-trade-up' : 'border-l-trade-down'
                      )}>
                        <div className="flex items-center gap-3 min-w-0">
                          <div className={cn(
                            'flex h-6 w-6 items-center justify-center rounded text-xs font-bold',
                            isBuy ? 'bg-trade-up/10 text-trade-up' : 'bg-trade-down/10 text-trade-down'
                          )}>
                            {isBuy ? 'B' : 'S'}
                          </div>
                          <div className="flex flex-col gap-0.5 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium truncate">{trade.symbol}</span>
                              {tag === 'open' && <Badge variant="outline" className="rounded-md text-[10px] border-blue-400 text-blue-400">{t('tradeTags.open')}</Badge>}
                              {tag === 'close' && <Badge variant="outline" className="rounded-md text-[10px] border-orange-400 text-orange-400">{t('tradeTags.close')}</Badge>}
                              {tag === 'add' && <Badge variant="outline" className="rounded-md text-[10px] border-emerald-400 text-emerald-400">{t('tradeTags.add')}</Badge>}
                              {tag === 'reduce' && <Badge variant="outline" className="rounded-md text-[10px] border-amber-400 text-amber-400">{t('tradeTags.reduce')}</Badge>}
                              {trade.isMaker && <Badge variant="outline" className="rounded-md text-[10px]">{t('tradeTags.maker')}</Badge>}
                              <span className="text-[10px] text-muted-foreground tabular-nums">{t('tradeTags.net', { position: netPos.toFixed(6) })}</span>
                            </div>
                            <span className="text-xs text-muted-foreground">{trade.exchange}</span>
                          </div>
                        </div>
                        <div className="text-right space-y-0.5 shrink-0">
                          <p className="text-sm font-mono">{trade.price} × {trade.quantity}</p>
                          <div className="flex items-center justify-end gap-3 text-xs text-muted-foreground">
                            {trade.quoteQuantity && parseFloat(trade.quoteQuantity) > 0 && (
                              <span>${parseFloat(trade.quoteQuantity).toFixed(2)}</span>
                            )}
                            <span>{trade.fee} {trade.feeCurrency}</span>
                            {trade.tradedAt && <span className="tabular-nums">{new Date(trade.tradedAt).toLocaleString()}</span>}
                          </div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </ScrollArea>
            ) : (
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                {isRunning ? t('noTrades') : t('startToSeeData')}
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
