'use client'

import { Link } from '@/i18n/navigation'
import { useTranslations } from 'next-intl'
import { useUserId } from '@/components/providers/user-id'
import {
  useUserStrategies,
  useBotTrades,
  useBotAssets,
  useBotSessions,
  useBotTradingVolume,
  useBotPnL,
  type BBGoTrade,
  type BBGoAsset,
} from '@/lib/bbgo/queries'
import { cn } from '@/lib/utils'
import { ErrorBoundary } from '@/components/error-boundary'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { AssetAllocationChart } from '@/components/dashboard/AssetAllocationChart'
import { TradingVolumeChart } from '@/components/dashboard/TradingVolumeChart'
import { PnlSummary } from '@/components/dashboard/PnlSummary'
import { PnlChart } from '@/components/dashboard/PnlChart'
import { EquityChart } from '@/components/dashboard/EquityChart'
import { useTradingMode } from '@/components/providers/trading-mode'
import {
  Activity,
  BarChart3,
  Wallet,
  ArrowUpRight,
  ArrowDownRight,
  Bot,
  Plus,
} from 'lucide-react'

export default function DashboardPage() {
  const t = useTranslations('Dashboard')
  const bt = useTranslations('Bots')
  const userId = useUserId()
  const { mode: globalMode } = useTradingMode()

  const { data: containersResp } = useUserStrategies(userId)
  const containers = containersResp?.containers ?? {}
  const activeContainer = containers[globalMode]
  const otherContainer = containers[globalMode === 'live' ? 'paper' : 'live']
  const isActive = activeContainer?.status === 'running'
  const anyActive = isActive || otherContainer?.status === 'running'
  const strategyCount = (activeContainer?.strategies?.length ?? 0) + (otherContainer?.strategies?.length ?? 0)

  const { data: tradesData } = useBotTrades(userId, undefined, undefined, globalMode)
  const { data: assetsData } = useBotAssets(userId, globalMode)
  const { data: sessionsData } = useBotSessions(userId, globalMode)
  const { data: volumeData } = useBotTradingVolume(userId, undefined, globalMode)
  const { data: pnlData } = useBotPnL(userId, undefined, undefined, globalMode)

  const trades = isActive ? (tradesData?.trades ?? []) : []
  const assets = isActive ? (assetsData?.assets ?? {}) : {}
  const sessionCount = isActive ? (sessionsData?.sessions?.length ?? 0) : 0

  const totalValue = Object.values(assets).reduce((sum, a: BBGoAsset) => {
    return sum + parseFloat(a.netAssetInUSD || '0')
  }, 0)

  return (
    <div className="space-y-8">
      <div>
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>
          <span className={cn(
            'inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-medium',
            globalMode === 'live' ? 'bg-blue-100 text-blue-700' : 'bg-amber-100 text-amber-700',
          )}>
            {bt(`mode.${globalMode}`)}
          </span>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          {anyActive
            ? t('strategyCount', { count: strategyCount })
            : t('noStrategies')}
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card className="rounded-xl">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-[13px] font-medium text-muted-foreground">{t('activeBots')}</p>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                <Bot className="h-4 w-4 text-primary" />
              </div>
            </div>
            <div className="mt-3 flex items-baseline gap-2">
              <span className="text-2xl font-semibold">{isActive ? 1 : 0}</span>
              <span className="text-sm text-muted-foreground">/ {strategyCount}</span>
            </div>
          </CardContent>
        </Card>

        <Card className="rounded-xl">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-[13px] font-medium text-muted-foreground">{t('sessions')}</p>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-chart-2/10">
                <Activity className="h-4 w-4 text-chart-2" />
              </div>
            </div>
            <p className="mt-3 text-2xl font-semibold font-mono">{sessionCount}</p>
          </CardContent>
        </Card>

        <Card className="rounded-xl">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-[13px] font-medium text-muted-foreground">{t('recentTrades')}</p>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-chart-3/10">
                <BarChart3 className="h-4 w-4 text-chart-3" />
              </div>
            </div>
            <p className="mt-3 text-2xl font-semibold font-mono">{trades.length}</p>
          </CardContent>
        </Card>

        <Card className="rounded-xl">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-[13px] font-medium text-muted-foreground">{t('portfolioValue')}</p>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                <Wallet className="h-4 w-4 text-primary" />
              </div>
            </div>
            <p className="mt-3 text-2xl font-semibold font-mono">
              {anyActive && totalValue > 0
                ? `$${totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
                : '—'}
            </p>
          </CardContent>
        </Card>
      </div>

      {isActive && (
        <ErrorBoundary>
          <div className="grid gap-4 md:grid-cols-2">
            <Card className="rounded-xl">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">{t('assetAllocation')}</CardTitle>
              </CardHeader>
              <CardContent>
                <AssetAllocationChart assets={assets} />
              </CardContent>
            </Card>

            <Card className="rounded-xl">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">{t('tradingVolume')}</CardTitle>
              </CardHeader>
              <CardContent>
                <TradingVolumeChart volumes={volumeData?.tradingVolumes ?? []} />
              </CardContent>
            </Card>
          </div>
        </ErrorBoundary>
      )}

      {isActive && pnlData && pnlData.totalTrades > 0 && (
        <ErrorBoundary>
          <Card className="rounded-xl">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">{t('pnlSummary')}</CardTitle>
            </CardHeader>
            <CardContent>
              <PnlSummary report={pnlData} />
            </CardContent>
          </Card>
        </ErrorBoundary>
      )}

      {isActive && trades.length > 0 && (
        <ErrorBoundary>
          <div className="grid gap-4 md:grid-cols-2">
            <Card className="rounded-xl">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">{t('pnlChart')}</CardTitle>
              </CardHeader>
              <CardContent>
                <PnlChart trades={trades} />
              </CardContent>
            </Card>

            <Card className="rounded-xl">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">{t('equityCurve')}</CardTitle>
              </CardHeader>
              <CardContent>
                <EquityChart assets={assets} />
              </CardContent>
            </Card>
          </div>
        </ErrorBoundary>
      )}

      {anyActive && strategyCount > 0 && (
        <Card className="rounded-xl">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">{t('strategies')}</CardTitle>
            <Link href="/user/bots">
              <Button variant="ghost" size="sm" className="text-xs text-primary">
                {t('manage')} <ArrowUpRight className="ml-1 h-3 w-3" />
              </Button>
            </Link>
          </CardHeader>
          <div className="divide-y">
            {[activeContainer, otherContainer].filter(Boolean).flatMap(uc => uc!.strategies.map(s => ({ ...s, containerStatus: uc!.status }))).map((s) => (
              <div key={s.id} className="flex items-center justify-between px-6 py-3.5">
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                    <Bot className="h-4 w-4 text-muted-foreground" />
                  </div>
                  <div>
                    <p className="text-sm font-medium">{s.name || s.strategy}</p>
                    <p className="text-xs text-muted-foreground">
                      {s.exchange} · {s.strategy} · {bt(`mode.${s.mode}`)}
                    </p>
                  </div>
                </div>
                <Badge
                  variant={s.containerStatus === 'running' ? 'default' : 'secondary'}
                  className={cn(
                    'rounded-full text-[11px] font-medium',
                    s.containerStatus === 'running' && 'bg-trade-up text-white hover:bg-trade-up'
                  )}
                >
                  {s.containerStatus === 'running' ? bt('strategyStatus.running') : bt('strategyStatus.idle')}
                </Badge>
              </div>
            ))}
          </div>
        </Card>
      )}

      {trades.length > 0 && (
        <Card className="rounded-xl">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">{t('recentTrades')}</CardTitle>
            <Link href={`/user/bots/${userId}?mode=${globalMode}`}>
              <Button variant="ghost" size="sm" className="text-xs text-primary">
                {t('viewAll')} <ArrowUpRight className="ml-1 h-3 w-3" />
              </Button>
            </Link>
          </CardHeader>
          <div className="divide-y">
            {trades.slice(0, 10).map((trade: BBGoTrade) => (
              <div key={trade.id} className="flex items-center justify-between px-6 py-3">
                <div className="flex items-center gap-3">
                  <div
                    className={cn(
                      'flex h-7 w-7 items-center justify-center rounded-full',
                      trade.side === 'BUY' ? 'bg-trade-up' : 'bg-trade-down'
                    )}
                  >
                    {trade.side === 'BUY' ? (
                      <ArrowDownRight className="h-3.5 w-3.5 text-trade-up" />
                    ) : (
                      <ArrowUpRight className="h-3.5 w-3.5 text-trade-down" />
                    )}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{trade.symbol}</span>
                      <Badge variant="secondary" className="rounded-md text-[10px] font-medium">
                        {trade.side}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">{trade.exchange}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm font-medium font-mono">
                    {trade.price} × {trade.quantity}
                  </p>
                  {trade.tradedAt && (
                    <p className="text-xs text-muted-foreground">
                      {new Date(trade.tradedAt).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {userId && !anyActive && strategyCount === 0 && (
        <Card className="rounded-xl">
          <CardContent className="flex flex-col items-center justify-center py-16">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 mb-4">
              <Plus className="h-6 w-6 text-primary" />
            </div>
            <p className="text-sm text-muted-foreground mb-4">{t('noStrategies')}</p>
            <Link href="/user/bots">
              <Button className="rounded-full px-6">{t('createStrategy')}</Button>
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
