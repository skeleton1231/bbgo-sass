'use client'

import { useTranslations } from 'next-intl'
import { useSupabaseFuturesPositions, useSupabaseMarginHistory } from '@/lib/bbgo/supabase-queries'
import { useUserId } from '@/components/providers/user-id'
import { useTradingMode } from '@/components/providers/trading-mode'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { TrendingUp, Landmark, ArrowDownToLine, ArrowUpFromLine, Percent, AlertTriangle } from 'lucide-react'

function num(s: string | undefined | null): number {
  if (!s) return 0
  const v = parseFloat(s)
  return isNaN(v) ? 0 : v
}

function pnlColor(v: number) {
  return v > 0 ? 'text-trade-up' : v < 0 ? 'text-trade-down' : ''
}

function pnlSign(v: number) {
  return v > 0 ? '+' : ''
}

export function FuturesMarginPanel() {
  const t = useTranslations('Futures')
  const userId = useUserId()
  const { mode } = useTradingMode()
  const { data: positionsData, isLoading: posLoading } = useSupabaseFuturesPositions(userId, { mode })
  const { data: marginData, isLoading: marginLoading } = useSupabaseMarginHistory(userId, { mode })

  const positions = positionsData ?? []
  const loans = marginData?.loans ?? []
  const repays = marginData?.repays ?? []
  const interests = marginData?.interests ?? []
  const liquidations = marginData?.liquidations ?? []

  if (posLoading && marginLoading) {
    return <div className="space-y-3"><Skeleton className="h-24" /><Skeleton className="h-24" /></div>
  }

  const hasFuturesData = positions.length > 0
  const hasMarginData = loans.length > 0 || repays.length > 0 || interests.length > 0 || liquidations.length > 0

  if (!hasFuturesData && !hasMarginData) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <Landmark className="h-10 w-10 mb-3 opacity-40" />
        <p className="text-sm">{t('noData')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {hasFuturesData && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <TrendingUp className="h-4 w-4" />
              {t('positionRisks')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="text-left py-1.5 pr-3">{t('symbol')}</th>
                    <th className="text-right py-1.5 pr-3">{t('leverage')}</th>
                    <th className="text-right py-1.5 pr-3">{t('entryPrice')}</th>
                    <th className="text-right py-1.5 pr-3">{t('markPrice')}</th>
                    <th className="text-right py-1.5 pr-3">{t('liqPrice')}</th>
                    <th className="text-right py-1.5 pr-3">{t('positionAmt')}</th>
                    <th className="text-right py-1.5 pr-3">{t('unrealizedPnl')}</th>
                    <th className="text-right py-1.5">{t('notional')}</th>
                  </tr>
                </thead>
                <tbody>
                  {positions.map((p) => {
                    const upnl = num(p.unrealized_pnl)
                    const liqRisk = num(p.liquidation_price) > 0 && num(p.mark_price) > 0
                      ? Math.abs(num(p.mark_price) - num(p.liquidation_price)) / num(p.mark_price) * 100
                      : 0
                    return (
                      <tr key={p.id} className="border-b last:border-0">
                        <td className="py-1.5 pr-3 font-medium">
                          {p.symbol}
                          <span className="ml-1 text-muted-foreground">{p.position_side}</span>
                        </td>
                        <td className="text-right py-1.5 pr-3 font-mono">{num(p.leverage)}x</td>
                        <td className="text-right py-1.5 pr-3 font-mono">{num(p.entry_price).toFixed(2)}</td>
                        <td className="text-right py-1.5 pr-3 font-mono">{num(p.mark_price).toFixed(2)}</td>
                        <td className="text-right py-1.5 pr-3 font-mono">
                          <span className={liqRisk > 0 && liqRisk < 10 ? 'text-trade-down' : ''}>
                            {num(p.liquidation_price).toFixed(2)}
                          </span>
                        </td>
                        <td className="text-right py-1.5 pr-3 font-mono">
                          {num(p.position_amount) > 0 ? (
                            <span className="text-blue-400">{num(p.position_amount).toFixed(4)} L</span>
                          ) : num(p.position_amount) < 0 ? (
                            <span className="text-rose-400">{Math.abs(num(p.position_amount)).toFixed(4)} S</span>
                          ) : (
                            '0.0000'
                          )}
                        </td>
                        <td className={`text-right py-1.5 pr-3 font-mono font-semibold ${pnlColor(upnl)}`}>
                          {pnlSign(upnl)}{upnl.toFixed(4)}
                        </td>
                        <td className="text-right py-1.5 font-mono">{num(p.notional).toFixed(2)}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}

      {hasMarginData && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Landmark className="h-4 w-4" />
              {t('marginHistory')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <MarginStat icon={<ArrowDownToLine className="h-4 w-4" />} label={t('loans')} value={loans.length} />
              <MarginStat icon={<ArrowUpFromLine className="h-4 w-4" />} label={t('repays')} value={repays.length} />
              <MarginStat icon={<Percent className="h-4 w-4" />} label={t('interests')} value={interests.length} />
              <MarginStat icon={<AlertTriangle className="h-4 w-4 text-trade-down" />} label={t('liquidations')} value={liquidations.length} />
            </div>

            {interests.length > 0 && (
              <div className="mt-4">
                <h4 className="text-xs font-semibold text-muted-foreground mb-2 uppercase tracking-wider">{t('recentInterests')}</h4>
                <div className="overflow-x-auto">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b text-muted-foreground">
                        <th className="text-left py-1.5 pr-3">{t('asset')}</th>
                        <th className="text-right py-1.5 pr-3">{t('principle')}</th>
                        <th className="text-right py-1.5 pr-3">{t('interest')}</th>
                        <th className="text-right py-1.5 pr-3">{t('rate')}</th>
                        <th className="text-right py-1.5">{t('time')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {interests.slice(0, 10).map((i) => (
                        <tr key={i.id} className="border-b last:border-0">
                          <td className="py-1.5 pr-3 font-medium">{i.asset}</td>
                          <td className="text-right py-1.5 pr-3 font-mono">{num(i.principle).toFixed(6)}</td>
                          <td className="text-right py-1.5 pr-3 font-mono">{num(i.interest).toFixed(6)}</td>
                          <td className="text-right py-1.5 pr-3 font-mono">{num(i.interest_rate).toFixed(6)}</td>
                          <td className="text-right py-1.5 text-muted-foreground">{new Date(i.time).toLocaleString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {liquidations.length > 0 && (
              <div className="mt-4">
                <h4 className="text-xs font-semibold text-trade-down mb-2 uppercase tracking-wider">{t('recentLiquidations')}</h4>
                <div className="overflow-x-auto">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b text-muted-foreground">
                        <th className="text-left py-1.5 pr-3">{t('symbol')}</th>
                        <th className="text-right py-1.5 pr-3">{t('side')}</th>
                        <th className="text-right py-1.5 pr-3">{t('price')}</th>
                        <th className="text-right py-1.5 pr-3">{t('quantity')}</th>
                        <th className="text-right py-1.5">{t('time')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {liquidations.slice(0, 10).map((l) => (
                        <tr key={l.id} className="border-b last:border-0">
                          <td className="py-1.5 pr-3 font-medium">{l.symbol}</td>
                          <td className="text-right py-1.5 pr-3">{l.side}</td>
                          <td className="text-right py-1.5 pr-3 font-mono">{num(l.price).toFixed(2)}</td>
                          <td className="text-right py-1.5 pr-3 font-mono">{num(l.quantity).toFixed(4)}</td>
                          <td className="text-right py-1.5 text-muted-foreground">{new Date(l.time).toLocaleString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function MarginStat({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="flex items-center gap-2 rounded-md border p-2">
      <div className="text-muted-foreground">{icon}</div>
      <div>
        <div className="text-lg font-semibold">{value}</div>
        <div className="text-xs text-muted-foreground">{label}</div>
      </div>
    </div>
  )
}
