'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { useUserStrategies } from '@/lib/bbgo/queries'
import { cn } from '@/lib/utils'

interface RecentTrade {
  id: string
  symbol: string
  side: string
  price: string
  quantity: string
  pnl: string | null
  created_at: string
  bot_id: string
}

export default function DashboardPage() {
  const t = useTranslations('Dashboard')
  const [userId, setUserId] = useState('')
  const [recentTrades, setRecentTrades] = useState<RecentTrade[]>([])

  const { data: userContainer } = useUserStrategies(userId)

  useEffect(() => {
    const load = async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data: userData } = await supabase.auth.getUser()
      if (!userData.user) return
      setUserId(userData.user.id)

      const { data } = await supabase
        .from('sync_trades')
        .select('id, symbol, side, price, quantity, pnl, created_at, bot_id')
        .eq('user_id', userData.user.id)
        .order('created_at', { ascending: false })
        .limit(10)
      if (data) setRecentTrades(data)
    }
    load()
  }, [])

  const isActive = userContainer?.status === 'running'
  const strategyCount = userContainer?.strategies?.length ?? 0

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('activeBots')}</p>
          <p className="text-2xl font-bold mt-1">
            {isActive ? 1 : 0}<span className="text-base font-normal text-muted-foreground"> / {strategyCount} strategies</span>
          </p>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('recentTrades')}</p>
          <p className="text-2xl font-bold mt-1">{recentTrades.length}</p>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <p className="text-sm text-muted-foreground">{t('todayPnl')}</p>
          <p className="text-2xl font-bold mt-1">--</p>
        </div>
      </div>

      {recentTrades.length > 0 && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">{t('recentTrades')}</h2>
          </div>
          <div className="divide-y">
            {recentTrades.map((trade) => (
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
                </div>
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <span>{trade.price}</span>
                  <span>x {trade.quantity}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {!userId && (
        <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
          Loading...
        </div>
      )}
    </div>
  )
}
