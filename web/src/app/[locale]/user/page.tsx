'use client'

import { Link } from '@/i18n/navigation'
import { useTranslations } from 'next-intl'
import { useUserId } from '@/components/providers/user-id'
import {
  useUserStrategies,
  type BBGoTrade,
  type BBGoAsset,
} from '@/lib/bbgo/queries'
import { useSupabaseTrades, useSupabasePnL, useSupabaseTradingVolume, useSupabaseBalances, tradeKey } from '@/lib/bbgo/supabase-queries'
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

  const { data: instancesResp } = useUserStrategies(userId)
  const allInstances = instancesResp?.instances ?? []
  const activeInstances = allInstances.filter((i) => i.mode === globalMode)
  const isActive = activeInstances.some((i) => i.status === 'running')
  const anyActive = allInstances.some((i) => i.status === 'running')
  const strategyCount = activeInstances.length

  // Same opts as useSupabaseTradingVolume's internal call — RQ dedupes by queryKey,
  // so this shares the 5000-row fetch instead of issuing a separate 200-row one.
  const { data: tradesData } = useSupabaseTrades(userId, { mode: globalMode, limit: 5000 })
  const { data: balancesData } = useSupabaseBalances(userId, { mode: globalMode })
  const { data: volumeData } = useSupabaseTradingVolume(userId, { mode: globalMode })
  const { data: pnlData } = useSupabasePnL(userId, { mode: globalMode })

  const trades = tradesData ?? []

  // Estimate BTC price from latest trade
  const latestBtcPrice = trades.find((t) => t.symbol === 'BTCUSDT')?.price
  const btcPrice = latestBtcPrice ? parseFloat(latestBtcPrice) : 0

  // Build assets from Supabase balances with USD estimation
  const rawBalances = balancesData ?? {}
  const assets: Record<string, BBGoAsset> = {}
  for (const [currency, b] of Object.entries(rawBalances)) {
    const total = parseFloat(b.available) + parseFloat(b.locked)
    if (total <= 0) continue
    const priceInUSD = currency === 'USDT' ? 1 : currency === 'BTC' ? btcPrice : 0
    const netAssetInUSD = (total * priceInUSD).toFixed(2)
    assets[currency] = {
      currency,
      total: total.toString(),
      available: b.available,
      lock: b.locked,
      borrowed: '0',
      netAsset: total.toString(),
      netAssetInUSD,
      netAssetInBTC: '0',
      priceInUSD: priceInUSD.toString(),
    }
  }

  const sessionCount = activeInstances.filter((i) => i.status === 'running').length

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
          {strategyCount > 0
            ? t('strategyCount', { count: strategyCount })
            : t('noStrategies')}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-4">
        <Card className="rounded-xl">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-[13px] font-medium text-muted-foreground">{t('activeBots')}</p>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                <Bot className="h-4 w-4 text-primary" />
              </div>
            </div>
            <div className="mt-3 flex items-baseline gap-2">
              <span className="text-2xl font-semibold">{strategyCount}</span>
              <span className="text-sm text-muted-foreground">{t('strategyCount', { count: strategyCount })}</span>
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
              {totalValue > 0
                ? `$${totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
                : '—'}
            </p>
          </CardContent>
        </Card>
      </div>

      {Object.keys(assets).length > 0 && (
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

      {pnlData && pnlData.totalTrades > 0 && (
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

      {pnlData && pnlData.totalTrades > 0 && (
        <ErrorBoundary>
          <div className="grid gap-4 md:grid-cols-2">
            <Card className="rounded-xl">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">{t('pnlChart')}</CardTitle>
              </CardHeader>
              <CardContent>
                <PnlChart dailyBreakdown={pnlData.dailyBreakdown} />
              </CardContent>
            </Card>

            <Card className="rounded-xl">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">{t('equityCurve')}</CardTitle>
              </CardHeader>
              <CardContent>
                <EquityChart pnlCurve={pnlData.pnlCurve} />
              </CardContent>
            </Card>
          </div>
        </ErrorBoundary>
      )}

      {strategyCount > 0 && (
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
            {activeInstances.map((inst) => {
              const isRunning = inst.status === 'running'
              return (
              <div key={inst.instance_id} className="flex items-center justify-between px-6 py-3.5">
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                    <Bot className="h-4 w-4 text-muted-foreground" />
                  </div>
                  <div>
                    <p className="text-sm font-medium">{inst.name || inst.strategy}</p>
                    <p className="text-xs text-muted-foreground">
                      {inst.exchange} · {inst.symbol}
                    </p>
                  </div>
                </div>
                <Badge
                  variant={isRunning ? 'default' : 'secondary'}
                  className={cn(
                    'rounded-full text-[11px] font-medium',
                    isRunning && 'bg-trade-up text-white hover:bg-trade-up'
                  )}
                >
                  {isRunning ? bt('strategyStatus.running') : bt('strategyStatus.idle')}
                </Badge>
              </div>
            )})}
          </div>
        </Card>
      )}

      {trades.length > 0 && (
        <Card className="rounded-xl">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">{t('recentTrades')}</CardTitle>
            <Link href={`/user/bots?mode=${globalMode}`}>
              <Button variant="ghost" size="sm" className="text-xs text-primary">
                {t('viewAll')} <ArrowUpRight className="ml-1 h-3 w-3" />
              </Button>
            </Link>
          </CardHeader>
          <div className="divide-y">
            {trades.slice(0, 10).map((trade: BBGoTrade) => (
              <div key={tradeKey(trade)} className="flex items-center justify-between px-6 py-3">
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
