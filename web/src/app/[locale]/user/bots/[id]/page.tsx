'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { useUserStrategies, useStartUser, useStopUser, useSupabaseOrders, useSupabaseTrades } from '@/lib/bbgo/queries'
import { cn } from '@/lib/utils'

export default function BotDetailPage() {
  const t = useTranslations('Bots')
  const router = useRouter()
  const [userId, setUserId] = useState('')

  useEffect(() => {
    const loadUser = async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data } = await supabase.auth.getUser()
      if (data.user) setUserId(data.user.id)
    }
    loadUser()
  }, [])

  const { data: userContainer, isLoading } = useUserStrategies(userId)
  const startUser = useStartUser()
  const stopUser = useStopUser()
  const { data: orders } = useSupabaseOrders(userId)
  const { data: trades } = useSupabaseTrades(userId)

  if (isLoading || !userId) {
    return <div className="text-muted-foreground">Loading...</div>
  }

  const strategies = userContainer?.strategies ?? []
  const status = userContainer?.status ?? 'stopped'

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <button onClick={() => router.back()} className="text-sm text-muted-foreground hover:text-foreground mb-2">
            &larr; Back to bots
          </button>
          <h1 className="text-2xl font-bold">Trading Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            {strategies.length} strategies · Container: bbgo-{userId.slice(0, 8)}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span
            className={cn(
              'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
              status === 'running' && 'bg-green-100 text-green-700',
              status === 'stopped' && 'bg-gray-100 text-gray-700',
              status === 'error' && 'bg-red-100 text-red-700'
            )}
          >
            {status}
          </span>
          {status === 'running' ? (
            <button
              onClick={() => stopUser.mutate(userId)}
              disabled={stopUser.isPending}
              className="rounded-md border px-4 py-2 text-sm hover:bg-muted disabled:opacity-50"
            >
              {t('stop')}
            </button>
          ) : (
            <button
              onClick={() => startUser.mutate(userId)}
              disabled={startUser.isPending}
              className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {t('start')}
            </button>
          )}
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">Recent Orders</h2>
          </div>
          {orders && orders.length > 0 ? (
            <div className="divide-y max-h-80 overflow-y-auto">
              {orders.map((order: Record<string, unknown>, i: number) => (
                <div key={i} className="flex items-center justify-between px-4 py-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className={cn(
                      'text-xs font-medium rounded px-1.5 py-0.5',
                      (order.side as string) === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                    )}>
                      {order.side as string}
                    </span>
                    <span>{order.symbol as string}</span>
                  </div>
                  <div className="text-muted-foreground">
                    {(order.price as string)} x {(order.quantity as string)}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="p-4 text-sm text-muted-foreground">No orders yet</div>
          )}
        </div>

        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">Recent Trades</h2>
          </div>
          {trades && trades.length > 0 ? (
            <div className="divide-y max-h-80 overflow-y-auto">
              {trades.map((trade: Record<string, unknown>, i: number) => (
                <div key={i} className="flex items-center justify-between px-4 py-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className={cn(
                      'text-xs font-medium rounded px-1.5 py-0.5',
                      (trade.side as string) === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                    )}>
                      {trade.side as string}
                    </span>
                    <span>{trade.symbol as string}</span>
                  </div>
                  <div className="text-muted-foreground">
                    {(trade.price as string)} x {(trade.quantity as string)}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="p-4 text-sm text-muted-foreground">No trades yet</div>
          )}
        </div>
      </div>

      {strategies.length > 0 && (
        <div className="rounded-lg border bg-card p-4">
          <h2 className="font-semibold mb-2">Strategies</h2>
          <div className="divide-y">
            {strategies.map((s) => (
              <div key={s.id} className="flex items-center justify-between py-2 text-sm">
                <div>
                  <p className="font-medium">{s.name || s.strategy}</p>
                  <p className="text-muted-foreground">{s.exchange} · {s.strategy} · {s.mode}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
