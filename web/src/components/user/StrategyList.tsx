'use client'

import { useTranslations } from 'next-intl'
import { useUserStrategies, useStartUser, useStopUser, useDeleteStrategy } from '@/lib/bbgo/queries'
import { cn } from '@/lib/utils'

export function StrategyList({ userId }: { userId: string }) {
  const t = useTranslations('Bots')
  const { data: userContainer, isLoading } = useUserStrategies(userId)
  const startUser = useStartUser()
  const stopUser = useStopUser()
  const deleteStrategy = useDeleteStrategy()

  if (isLoading) {
    return <div className="text-muted-foreground">Loading...</div>
  }

  const strategies = userContainer?.strategies ?? []
  const status = userContainer?.status ?? 'stopped'

  if (strategies.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
        No strategies yet. Create one to get started.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between rounded-lg border bg-card p-4">
        <div>
          <p className="font-medium">User Container</p>
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
              className="rounded-md border px-3 py-1 text-sm hover:bg-muted disabled:opacity-50"
            >
              {t('stop')}
            </button>
          ) : (
            <button
              onClick={() => startUser.mutate(userId)}
              disabled={startUser.isPending}
              className="rounded-md bg-primary px-3 py-1 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {t('start')}
            </button>
          )}
        </div>
      </div>

      {strategies.map((s) => (
        <div key={s.id} className="flex items-center justify-between rounded-lg border bg-card p-4">
          <div>
            <p className="font-medium">{s.name || s.strategy}</p>
            <p className="text-sm text-muted-foreground">
              {s.exchange} · {s.strategy} · {s.mode}
            </p>
          </div>
          <button
            onClick={() => deleteStrategy.mutate({ userId, strategyId: s.id })}
            disabled={deleteStrategy.isPending}
            className="rounded-md border border-destructive px-3 py-1 text-sm text-destructive hover:bg-destructive hover:text-destructive-foreground disabled:opacity-50"
          >
            Remove
          </button>
        </div>
      ))}
    </div>
  )
}
