'use client'

import { Link } from '@/i18n/navigation'
import { useTranslations } from 'next-intl'
import { useUserStrategies, useStartUser, useStopUser, useDeleteStrategy } from '@/lib/bbgo/queries'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

export function StrategyList({ userId }: { userId: string }) {
  const t = useTranslations('Bots')
  const { data: userContainer, isLoading } = useUserStrategies(userId)
  const startUser = useStartUser()
  const stopUser = useStopUser()
  const deleteStrategy = useDeleteStrategy()

  if (isLoading) {
    return <div className="text-muted-foreground">{t('loading')}</div>
  }

  const strategies = userContainer?.strategies ?? []
  const status = userContainer?.status ?? 'stopped'

  if (strategies.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
        {t('noStrategies')}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between rounded-lg border bg-card p-4">
        <div>
          <p className="font-medium">{t('userContainer')}</p>
          <p className="text-sm text-muted-foreground">
            {t('strategiesCount', { count: strategies.length })} · {t('containerName', { id: userId.slice(0, 8) })}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {status === 'running' && (
            <Link
              href={`/user/bots/${userId}`}
              className="rounded-md border px-3 py-1 text-sm hover:bg-muted"
            >
              {t('dashboard')}
            </Link>
          )}
          <span
            className={cn(
              'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
              status === 'running' && 'bg-green-100 text-green-700',
              status === 'stopped' && 'bg-gray-100 text-gray-700',
              status === 'error' && 'bg-red-100 text-red-700'
            )}
          >
            {t(`status.${status}`)}
          </span>
          {status === 'running' ? (
            <button
              onClick={() => stopUser.mutate(userId, { onError: (err) => toast.error(err.message) })}
              disabled={stopUser.isPending}
              className="rounded-md border px-3 py-1 text-sm hover:bg-muted disabled:opacity-50"
            >
              {t('stop')}
            </button>
          ) : (
            <button
              onClick={() => startUser.mutate(userId, { onError: (err) => toast.error(err.message) })}
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
}
