'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import {
  useUserStrategies,
  useBotTrades,
  useBotAssets,
  useBotSessions,
  type BBGoTrade,
  type BBGoAsset,
} from '@/lib/bbgo/queries'
import { cn } from '@/lib/utils'

export default function DashboardPage() {
  const t = useTranslations('Dashboard')
  const bt = useTranslations('Bots')
  const [userId, setUserId] = useState('')

  useEffect(() => {
    const load = async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data: userData } = await supabase.auth.getUser()
      if (userData.user) setUserId(userData.user.id)
    }
    load()
  }, [])

  const { data: userContainer } = useUserStrategies(userId)
  const { data: tradesData } = useBotTrades(userId)
  const { data: assetsData } = useBotAssets(userId)
  const { data: sessionsData } = useBotSessions(userId)

  const trades = tradesData?.trades ?? []
  const assets = assetsData?.assets ?? {}
  const isActive = userContainer?.status === 'running'
  const strategyCount = userContainer?.strategies?.length ?? 0
  const sessionCount = sessionsData?.sessions?.length ?? 0

  const totalValue = Object.values(assets).reduce((sum, a: BBGoAsset) => {
    return sum + parseFloat(a.netAssetInUSD || '0')
  }, 0)

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      <div className="grid gap-4 md:grid-cols-4">
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('activeBots')}</p>
          <p className="text-2xl font-bold mt-1">
            {isActive ? 1 : 0}
            <span className="text-base font-normal text-muted-foreground"> / {strategyCount} strategies</span>
          </p>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('sessions')}</p>
          <p className="text-2xl font-bold mt-1">{sessionCount}</p>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('recentTrades')}</p>
          <p className="text-2xl font-bold mt-1">{trades.length}</p>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('portfolioValue')}</p>
          <p className="text-2xl font-bold mt-1">
            {isActive && totalValue > 0
              ? `$${totalValue.toFixed(2)}`
              : '--'}
          </p>
        </div>
      </div>

      {isActive && strategyCount > 0 && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b flex items-center justify-between">
            <h2 className="font-semibold">{t('strategies')}</h2>
            <Link href="/user/bots" className="text-sm text-primary hover:underline">
              {t('manage')}
            </Link>
          </div>
          <div className="divide-y">
            {userContainer!.strategies.map((s) => (
              <div key={s.id} className="flex items-center justify-between px-4 py-3">
                <div>
                  <p className="text-sm font-medium">{s.name || s.strategy}</p>
                  <p className="text-xs text-muted-foreground">{s.exchange} · {s.strategy} · {bt(`mode.${s.mode}`)}</p>
                </div>
                <span className={cn(
                  'text-xs font-medium rounded-full px-2 py-0.5',
                  isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                )}>
                  {isActive ? bt('strategyStatus.running') : bt('strategyStatus.idle')}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {trades.length > 0 && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">{t('recentTrades')}</h2>
          </div>
          <div className="divide-y">
            {trades.slice(0, 10).map((trade: BBGoTrade) => (
              <div key={trade.id} className="flex items-center justify-between px-4 py-3">
                <div className="flex items-center gap-3">
                  <span
                    className={cn(
                      'text-xs font-medium rounded px-1.5 py-0.5',
                      trade.side === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                    )}
                  >
                    {trade.side}
                  </span>
                  <span className="text-sm font-medium">{trade.symbol}</span>
                  <span className="text-xs text-muted-foreground">{trade.exchange}</span>
                </div>
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <span>{trade.price} x {trade.quantity}</span>
                  {trade.tradedAt && (
                    <span className="text-xs">{new Date(trade.tradedAt).toLocaleString()}</span>
                  )}
                </div>
              </div>
            ))}
          </div>
          <div className="p-3 border-t text-center">
            <Link href={`/user/bots/${userId}`} className="text-sm text-primary hover:underline">
              {t('viewAll')}
            </Link>
          </div>
        </div>
      )}

      {!userId && (
        <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
          Loading...
        </div>
      )}

      {userId && !isActive && strategyCount === 0 && (
        <div className="rounded-lg border bg-card p-8 text-center">
          <p className="text-muted-foreground mb-4">{t('noStrategies')}</p>
          <Link
            href="/user/bots"
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            {t('createStrategy')}
          </Link>
        </div>
      )}
    </div>
  )
}
