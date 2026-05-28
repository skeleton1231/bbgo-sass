'use client'

import { useState, useEffect, useMemo, useCallback } from 'react'
import { useRouter } from '@/i18n/navigation'
import { useParams, useSearchParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import dynamic from 'next/dynamic'
import {
  useUserStrategies,
  useStartUser,
  useStopUser,
  useBotSessions,
  useBotOpenOrders,
  useBotClosedOrders,
  useBotTrades,
  useBotSessionBalances,
  useBotStrategiesState,
  useBotPing,
  useContainerLogs,
  useBotPnL,
  type BBGoOrder,
  type BBGoTrade,
  type BBGoBalance,
} from '@/lib/bbgo/queries'
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
import type { TradeMarker, OrderLevel } from '@/components/chart/CandlestickChart'
import {
  ArrowLeft,
  ArrowUpRight,
  ArrowDownRight,
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

const CandlestickChart = dynamic(
  () => import('@/components/chart/CandlestickChart').then((m) => ({ default: m.CandlestickChart })),
  { ssr: false, loading: () => <div className="h-[450px] animate-pulse rounded-lg bg-muted" /> }
)

const DepthChart = dynamic(
  () => import('@/components/chart/DepthChart').then((m) => ({ default: m.DepthChart })),
  { ssr: false, loading: () => <div className="h-[300px] animate-pulse rounded-lg bg-muted" /> }
)

const KLINE_INTERVALS = [
  { key: '1m', label: '1m' },
  { key: '5m', label: '5m' },
  { key: '15m', label: '15m' },
  { key: '1h', label: '1H' },
  { key: '4h', label: '4H' },
  { key: '1d', label: '1D' },
] as const

export default function BotDetailPage() {
  const t = useTranslations('Bots')
  const router = useRouter()
  const params = useParams<{ id: string }>()
  const searchParams = useSearchParams()
  const userId = params.id
  const rawMode = searchParams.get('mode')
  const { mode: globalMode } = useTradingMode()
  const mode: 'live' | 'paper' = rawMode === 'paper' ? 'paper' : rawMode === 'live' ? 'live' : globalMode
  const [activeSession, setActiveSession] = useState<string>('')
  const [klineInterval, setKlineInterval] = useState('1h')
  const [depthData, setDepthData] = useState<{ bids: Array<{ price: number; volume: number }>; asks: Array<{ price: number; volume: number }> }>({ bids: [], asks: [] })

  const { data: containersResp, isLoading, isError } = useUserStrategies(userId)
  const startUser = useStartUser()
  const stopUser = useStopUser()

  const userContainer = containersResp?.containers?.[mode]

  const { data: sessionsData } = useBotSessions(userId, mode)
  const sessions = sessionsData?.sessions ?? []
  const firstSession = sessions[0]?.name ?? ''

  useEffect(() => {
    if (!activeSession && firstSession) setActiveSession(firstSession)
  }, [firstSession, activeSession])

  const isRunning = userContainer?.status === 'running'
  const { data: openOrdersData } = useBotOpenOrders(userId, activeSession, mode)
  const { data: closedOrdersData } = useBotClosedOrders(userId, undefined, undefined, mode)
  const { data: tradesData } = useBotTrades(userId, undefined, undefined, mode)
  const { data: balancesData } = useBotSessionBalances(userId, activeSession, mode)
  const { data: strategyStatesData } = useBotStrategiesState(userId, mode)
  const { data: pingData } = useBotPing(userId, mode)
  const { data: logsData } = useContainerLogs(userId, '100', mode)
  const { data: pnlData } = useBotPnL(userId, undefined, undefined, mode)

  const activeExchange = sessions.find((s) => s.name === activeSession)?.exchange ?? ''
  const activeSymbol = (userContainer?.strategies?.find(s => s.mode === mode)?.config?.symbol ?? userContainer?.strategies?.[0]?.config?.symbol ?? '') as string

  const { candles, isLoading: klinesLoading } = useKlineData({
    userId,
    exchange: activeExchange,
    symbol: activeSymbol,
    interval: klineInterval,
    enabled: isRunning && !!activeExchange && !!activeSymbol,
  })

  const tradeMarkers: TradeMarker[] = useMemo(() => {
    if (!tradesData?.trades) return []
    return tradesData.trades
      .filter((tr) => !activeSymbol || tr.symbol === activeSymbol)
      .slice(0, 200)
      .map((tr) => ({
        time: Math.floor(new Date(tr.tradedAt).getTime() / 1000) as TradeMarker["time"],
        side: tr.side as 'BUY' | 'SELL',
        price: parseFloat(tr.price),
        quantity: parseFloat(tr.quantity),
      }))
  }, [tradesData?.trades, activeSymbol])

  const orderLevels: OrderLevel[] = useMemo(() => {
    if (!openOrdersData?.orders) return []
    return openOrdersData.orders
      .filter((o) => !activeSymbol || o.symbol === activeSymbol)
      .map((o) => ({
        price: parseFloat(o.price),
        side: o.side as 'BUY' | 'SELL',
        quantity: o.executedQuantity || o.quantity,
      }))
  }, [openOrdersData?.orders, activeSymbol])

  const handleWSMessage = useCallback((msg: { type: string; data: { channel?: string; depth?: { bids: Array<{ price: string; volume: string }>; asks: Array<{ price: string; volume: string }> } } }) => {
    if (msg.type !== 'market' || !msg.data.depth) return
    setDepthData({
      bids: msg.data.depth.bids.slice(0, 20).map((b) => ({ price: parseFloat(b.price), volume: parseFloat(b.volume) })),
      asks: msg.data.depth.asks.slice(0, 20).map((a) => ({ price: parseFloat(a.price), volume: parseFloat(a.volume) })),
    })
  }, [])

  const { connected: wsConnected } = useMarketData({
    userId,
    enabled: isRunning,
    onMessage: handleWSMessage,
  })

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => <Skeleton key={i} className="h-20 rounded-xl" />)}
        </div>
      </div>
    )
  }

  if (isError) {
    return (
      <Card className="rounded-xl border-destructive/50">
        <CardContent className="flex flex-col items-center py-12">
          <AlertCircle className="h-8 w-8 text-destructive mb-3" />
          <p className="text-sm text-destructive">{t('errorLoading')}</p>
        </CardContent>
      </Card>
    )
  }

  const strategies = userContainer?.strategies ?? []
  const status = userContainer?.status ?? 'stopped'
  const openOrders = openOrdersData?.orders ?? []
  const closedOrders = closedOrdersData?.orders ?? []
  const trades = tradesData?.trades ?? []
  const balances = balancesData?.balances ?? {}
  const liveStrategies = strategyStatesData?.strategies ?? []
  const botReachable = isRunning && pingData?.status === 'ok'
  const nonZeroBalances = Object.entries(balances).filter(
    ([, b]) => parseFloat(b.available) > 0 || parseFloat(b.locked) > 0
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <button
            onClick={() => router.back()}
            className="inline-flex items-center gap-1 text-[13px] text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            {t('backToBots')}
          </button>
          <h1 className="text-2xl font-semibold tracking-tight">{t('tradingDashboard')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('strategiesCount', { count: strategies.length })} · {t('containerName', { id: userId.slice(0, 8) })} · {t(`mode.${mode}`)}
          </p>
        </div>

        <div className="flex items-center gap-3">
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
              disabled={startUser.isPending}
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
              onClick={() => setActiveSession(s.name)}
              className="rounded-full text-xs"
            >
              {s.exchange || s.name}
            </Button>
          ))}
        </div>
      )}

      {botReachable && pnlData && pnlData.totalTrades > 0 && (
        <div className="grid gap-4 md:grid-cols-4">
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
                  ({pnlData.winningTrades}W / {pnlData.losingTrades}L)
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
                      Pos: {s.openPosition.toFixed(6)}
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
        <TabsList className="bg-muted/50 p-1 rounded-lg">
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
          <ErrorBoundary>
          <Card className="rounded-xl">
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium">
                  {activeSymbol || 'Price Chart'} · {activeExchange}
                </CardTitle>
                <div className="flex gap-1">
                  {KLINE_INTERVALS.map((iv) => (
                    <button
                      key={iv.key}
                      onClick={() => setKlineInterval(iv.key)}
                      className={cn(
                        'rounded-md px-2 py-0.5 text-xs font-medium transition-colors',
                        klineInterval === iv.key
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-muted-foreground hover:bg-muted/80'
                      )}
                    >
                      {iv.label}
                    </button>
                  ))}
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {!activeSymbol ? (
                <div className="flex h-[450px] items-center justify-center text-sm text-muted-foreground">
                  {botReachable ? t('noSymbolForChart') : t('startToSeeData')}
                </div>
              ) : (
                <CandlestickChart
                  candles={candles}
                  tradeMarkers={tradeMarkers}
                  orderLevels={orderLevels}
                  height={450}
                  isLoading={klinesLoading}
                />
              )}
            </CardContent>
          </Card>
          </ErrorBoundary>
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
                  {trades.map((trade: BBGoTrade) => (
                    <div key={trade.id} className="flex items-center justify-between px-6 py-3">
                      <div className="flex items-center gap-3">
                        <div className={cn(
                          'flex h-7 w-7 items-center justify-center rounded-full',
                          trade.side === 'BUY' ? 'bg-trade-up' : 'bg-trade-down'
                        )}>
                          {trade.side === 'BUY'
                            ? <ArrowDownRight className="h-3.5 w-3.5 text-trade-up" />
                            : <ArrowUpRight className="h-3.5 w-3.5 text-trade-down" />}
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">{trade.symbol}</span>
                            <Badge variant="secondary" className="rounded-md text-[10px]">{trade.side}</Badge>
                            {trade.isMaker && <Badge variant="outline" className="rounded-md text-[10px]">{t('maker')}</Badge>}
                          </div>
                          <p className="text-xs text-muted-foreground">{trade.exchange}</p>
                        </div>
                      </div>
                      <div className="text-right space-y-0.5">
                        <p className="text-sm font-mono">{trade.price} × {trade.quantity}</p>
                        <div className="flex items-center justify-end gap-3 text-xs text-muted-foreground">
                          {trade.quoteQuantity && parseFloat(trade.quoteQuantity) > 0 && (
                            <span>${parseFloat(trade.quoteQuantity).toFixed(2)}</span>
                          )}
                          <span>{trade.fee} {trade.feeCurrency}</span>
                          {trade.tradedAt && <span>{new Date(trade.tradedAt).toLocaleString()}</span>}
                        </div>
                      </div>
                    </div>
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

        <TabsContent value="strategies">
          <Card className="rounded-xl">
            {strategies.length > 0 ? (
              <div className="divide-y">
                {strategies.map((s) => {
                  const liveState = liveStrategies.find((ls) => ls.strategy === s.strategy)
                  return (
                    <div key={s.id} className="px-6 py-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                            <Bot className="h-4 w-4 text-muted-foreground" />
                          </div>
                          <div>
                            <p className="text-sm font-medium">{s.name || s.strategy}</p>
                            <p className="text-xs text-muted-foreground">
                              {s.exchange}{s.crossExchange ? ` (${t('crossExchange')})` : ''} · {s.strategy} · {t(`mode.${s.mode}`)}
                            </p>
                          </div>
                        </div>
                        {isRunning && (
                          <Badge
                            variant={liveState ? 'default' : 'secondary'}
                            className={cn('rounded-full text-[11px]', liveState && 'bg-trade-up text-white hover:bg-trade-up')}
                          >
                            {liveState ? t('strategyStatus.running') : t('strategyStatus.idle')}
                          </Badge>
                        )}
                      </div>
                      {liveState && Object.keys(liveState).length > 1 && (
                        <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 pl-11">
                          {Object.entries(liveState)
                            .filter(([k]) => k !== 'strategy')
                            .slice(0, 6)
                            .map(([key, val]) => (
                              <span key={key} className="text-xs text-muted-foreground">
                                {key}: <span className="text-foreground font-mono">{typeof val === 'object' && val !== null ? JSON.stringify(val) : String(val)}</span>
                              </span>
                            ))}
                        </div>
                      )}
                    </div>
                  )
                })}
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

interface OrderRowProps {
  order: BBGoOrder
  showStatus?: boolean
  showTime?: boolean
}

function OrderRow({ order, showStatus, showTime }: OrderRowProps) {
  return (
    <div className="flex items-center justify-between px-6 py-3">
      <div className="flex items-center gap-3">
        <div className={cn(
          'flex h-7 w-7 items-center justify-center rounded-full',
          order.side === 'BUY' ? 'bg-trade-up' : 'bg-trade-down'
        )}>
          {order.side === 'BUY'
            ? <ArrowDownRight className="h-3.5 w-3.5 text-trade-up" />
            : <ArrowUpRight className="h-3.5 w-3.5 text-trade-down" />}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{order.symbol}</span>
          <Badge variant="secondary" className="rounded-md text-[10px]">{order.orderType}</Badge>
        </div>
      </div>
      <div className="flex items-center gap-4">
        <span className="text-sm text-muted-foreground font-mono">
          {order.price} × {order.executedQuantity || order.quantity}
        </span>
        {showStatus && order.status && (
          <Badge variant="outline" className={cn(
            'rounded-full text-[10px]',
            order.status === 'Filled' && 'border-trade-up/30 text-trade-up',
            order.status === 'Canceled' && 'border-border text-muted-foreground'
          )}>
            {order.status}
          </Badge>
        )}
        {showTime && order.creationTime && (
          <span className="text-xs text-muted-foreground">{new Date(order.creationTime).toLocaleString()}</span>
        )}
      </div>
    </div>
  )
}
