'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { TrendingUp, TrendingDown, ShieldAlert } from 'lucide-react'
import type { LatestPosition } from '@/lib/bbgo/supabase-queries'
import type { FuturesPositionRisk } from '@/lib/bbgo/manager'

interface PositionCardProps {
  position?: LatestPosition | null
  futuresRisk?: FuturesPositionRisk | null
  unrealized?: { unrealizedPnl: number; unrealizedPnlPct: number } | null
}

function num(s: string | undefined | null): number {
  if (!s) return 0
  const v = parseFloat(s)
  return isNaN(v) ? 0 : v
}

function pnlColor(v: number) {
  return v > 0 ? 'text-trade-up' : v < 0 ? 'text-trade-down' : ''
}

function pnlSign(v: number) {
  return v >= 0 ? '+' : ''
}

export function PositionCard({ position, futuresRisk, unrealized }: PositionCardProps) {
  const t = useTranslations('Bots')
  const isFutures = !!futuresRisk && Math.abs(num(futuresRisk.position_amount)) > 0

  if (isFutures) {
    return <FuturesPositionCard risk={futuresRisk!} />
  }

  if (!position || position.isClosed) return null
  return <SpotPositionCard position={position} unrealized={unrealized} />
}

function SpotPositionCard({
  position,
  unrealized,
}: {
  position: LatestPosition
  unrealized?: { unrealizedPnl: number; unrealizedPnlPct: number } | null
}) {
  const t = useTranslations('Bots')

  return (
    <Card className="rounded-xl">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium">{t('pnl.currentPosition')}</CardTitle>
      </CardHeader>
      <div className="px-6 pb-4 flex flex-wrap gap-x-8 gap-y-2 text-sm">
        <div>
          <span className="text-muted-foreground">{t('pnl.avgCost')}: </span>
          <span className="font-mono font-medium">{position.averageCost.toFixed(4)}</span>
        </div>
        <div>
          <span className="text-muted-foreground">
            {position.isLong
              ? t('pnl.positionLong', { qty: Math.abs(position.base).toFixed(6), price: position.averageCost.toFixed(4) })
              : position.isShort
                ? t('pnl.positionShort', { qty: Math.abs(position.base).toFixed(6), price: position.averageCost.toFixed(4) })
                : t('pnl.positionClosed')}
          </span>
        </div>
        {unrealized && (unrealized.unrealizedPnl !== 0 || unrealized.unrealizedPnlPct !== 0) && (
          <div>
            <span className="text-muted-foreground">{t('pnl.unrealized')}: </span>
            <span className={cn('font-mono font-medium', pnlColor(unrealized.unrealizedPnl))}>
              {pnlSign(unrealized.unrealizedPnl)}{unrealized.unrealizedPnl.toFixed(4)}
              <span className="ml-1 text-xs">({pnlSign(unrealized.unrealizedPnlPct)}{unrealized.unrealizedPnlPct.toFixed(2)}%)</span>
            </span>
          </div>
        )}
      </div>
    </Card>
  )
}

const DIRECTION_STYLES = {
  long: { badge: 'border-blue-400 text-blue-400 bg-blue-400/10', icon: TrendingUp },
  short: { badge: 'border-rose-400 text-rose-400 bg-rose-400/10', icon: TrendingDown },
}

function FuturesPositionCard({ risk }: { risk: FuturesPositionRisk }) {
  const t = useTranslations('Bots')

  const amount = num(risk.position_amount)
  const direction = amount > 0 ? 'long' as const : amount < 0 ? 'short' as const : null
  if (!direction) return null

  const upnl = num(risk.unrealized_pnl)
  const leverage = num(risk.leverage)
  const liqPrice = num(risk.liquidation_price)
  const markPrice = num(risk.mark_price)
  const entryPrice = num(risk.entry_price)
  const notional = num(risk.notional)

  const dirStyle = DIRECTION_STYLES[direction]
  const DirIcon = dirStyle.icon
  const dirLabel = direction === 'long' ? t('positionAction.open_long') : t('positionAction.open_short')

  const liqRisk = liqPrice > 0 && markPrice > 0
    ? Math.abs(markPrice - liqPrice) / markPrice * 100
    : 0

  return (
    <Card className="rounded-xl">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">{t('pnl.currentPosition')}</CardTitle>
          <div className={cn('flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium', dirStyle.badge)}>
            <DirIcon className="h-3 w-3" />
            {dirLabel}
          </div>
        </div>
      </CardHeader>
      <div className="px-6 pb-4 flex flex-wrap gap-x-8 gap-y-2 text-sm">
        <div>
          <span className="text-muted-foreground">{risk.symbol}</span>
          <span className="mx-2 text-muted-foreground">·</span>
          <span className="font-mono font-semibold">{leverage}x</span>
        </div>
        <div>
          <span className="text-muted-foreground">{t('pnl.entry')}: </span>
          <span className="font-mono font-medium">{entryPrice.toFixed(2)}</span>
        </div>
        <div>
          <span className="text-muted-foreground">{t('pnl.mark')}: </span>
          <span className="font-mono font-medium">{markPrice.toFixed(2)}</span>
        </div>
        <div>
          <span className="text-muted-foreground">{t('pnl.qty')}: </span>
          <span className="font-mono font-medium">{Math.abs(amount).toFixed(4)}</span>
        </div>
        {liqPrice > 0 && (
          <div>
            <span className="text-muted-foreground">{t('pnl.liqPrice')}: </span>
            <span className={cn('font-mono font-medium', liqRisk > 0 && liqRisk < 10 && 'text-trade-down')}>
              {liqPrice.toFixed(2)}
            </span>
            {liqRisk > 0 && liqRisk < 20 && (
              <ShieldAlert className={cn('inline h-3 w-3 ml-1', liqRisk < 10 ? 'text-trade-down' : 'text-yellow-500')} />
            )}
          </div>
        )}
        <div>
          <span className="text-muted-foreground">{t('pnl.unrealized')}: </span>
          <span className={cn('font-mono font-medium', pnlColor(upnl))}>
            {pnlSign(upnl)}{upnl.toFixed(4)}
          </span>
        </div>
        <div>
          <span className="text-muted-foreground">{t('pnl.notional')}: </span>
          <span className="font-mono font-medium">${notional.toFixed(2)}</span>
        </div>
      </div>
    </Card>
  )
}
