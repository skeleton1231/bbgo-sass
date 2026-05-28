'use client'

import { Link } from '@/i18n/navigation'
import { useTranslations } from 'next-intl'
import { useUserStrategies, useStartUser, useStopUser, useDeleteStrategy } from '@/lib/bbgo/queries'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

export function StrategyList({ userId, activeMode }: { userId: string; activeMode?: 'live' | 'paper' }) {
  const t = useTranslations('Bots')
  const { data: containersResp, isLoading } = useUserStrategies(userId)
  const startUser = useStartUser()
  const stopUser = useStopUser()
  const deleteStrategy = useDeleteStrategy()

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2].map((i) => (
          <div key={i} className="rounded-lg border bg-card p-4">
            <div className="flex items-center justify-between">
              <div className="space-y-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-48" />
              </div>
              <Skeleton className="h-8 w-20 rounded-md" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  const containers = containersResp?.containers ?? {}
  const allStrategies = [
    ...(containers.live?.strategies ?? []).map(s => ({ ...s, mode: 'live' as const })),
    ...(containers.paper?.strategies ?? []).map(s => ({ ...s, mode: 'paper' as const })),
  ]

  if (allStrategies.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
        {t('noStrategies')}
      </div>
    )
  }

  const sortedModes = (['live', 'paper'] as Array<'live' | 'paper'>).sort((a, b) => {
    if (a === activeMode) return -1
    if (b === activeMode) return 1
    return 0
  })

  return (
    <div className="space-y-3">
      {sortedModes.map((mode) => {
        const uc = containers[mode]
        if (!uc) return null
        const strategies = uc.strategies ?? []
        const status = uc.status ?? 'stopped'
        const isHighlighted = mode === activeMode

        return (
          <div key={mode} className={cn('space-y-2', !isHighlighted && 'opacity-60')}>
            <div className={cn(
              'flex items-center justify-between rounded-lg border bg-card p-4 transition-all',
              isHighlighted && 'ring-1 ring-primary/20',
            )}>
              <div>
                <div className="flex items-center gap-2">
                  <p className="font-medium">{t(`mode.${mode}`)} {t('userContainer')}</p>
                  <span className={cn(
                    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                    mode === 'live' ? 'bg-blue-100 text-blue-700' : 'bg-amber-100 text-amber-700',
                  )}>
                    {t(`mode.${mode}`)}
                  </span>
                </div>
                <p className="text-sm text-muted-foreground">
                  {t('strategiesCount', { count: strategies.length })} · {t('containerName', { id: userId.slice(0, 8) })}
                  {mode === 'paper' && `-${t('mode.paper')}`}
                </p>
              </div>
              <div className="flex items-center gap-3">
                {status === 'running' && (
                  <Link
                    href={`/user/bots/${userId}?mode=${mode}`}
                    className="rounded-md border px-3 py-1 text-sm hover:bg-muted"
                  >
                    {t('dashboard')}
                  </Link>
                )}
                <span
                  className={cn(
                    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                    status === 'running' && 'bg-green-100 text-green-700',
                    status === 'starting' && 'bg-yellow-100 text-yellow-700',
                    status === 'stopped' && 'bg-gray-100 text-gray-700',
                    status === 'error' && 'bg-red-100 text-red-700',
                  )}
                >
                  {t(`status.${status}`)}
                </span>
                {status === 'running' ? (
                  <button
                    onClick={() => stopUser.mutate(
                      { userId, mode },
                      { onError: (err) => toast.error(err.message) },
                    )}
                    disabled={stopUser.isPending}
                    className="rounded-md border px-3 py-1 text-sm hover:bg-muted disabled:opacity-50"
                  >
                    {t('stop')}
                  </button>
                ) : status === 'starting' ? (
                  <button
                    disabled
                    className="rounded-md bg-primary px-3 py-1 text-sm text-primary-foreground opacity-50 cursor-not-allowed"
                  >
                    {t('start')}
                  </button>
                ) : (
                  <button
                    onClick={() => startUser.mutate(
                      { userId, mode },
                      { onError: (err) => toast.error(err.message) },
                    )}
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
                    {s.exchange} · {s.strategy} · {t(`mode.${s.mode}`)}
                  </p>
                </div>
                <button
                  onClick={() => {
                    if (confirm(t('removeConfirm'))) {
                      deleteStrategy.mutate({ userId, strategyId: s.id }, { onError: (err) => toast.error(err.message) })
                    }
                  }}
                  disabled={deleteStrategy.isPending}
                  className="rounded-md border border-destructive px-3 py-1 text-sm text-destructive hover:bg-destructive hover:text-destructive-foreground disabled:opacity-50"
                >
                  {t('remove')}
                </button>
              </div>
            ))}
          </div>
        )
      })}
    </div>
  )
}
