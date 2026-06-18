'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { Link } from '@/i18n/navigation'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Plus, Play, Square, Trash2, Bot as BotIcon, AlertCircle } from 'lucide-react'
import { useUserId } from '@/components/providers/user-id'
import { useTradingMode } from '@/components/providers/trading-mode'
import { useBotList, useStartInstance, useStopInstance, useDeleteStrategy } from '@/lib/bbgo/queries'
import type { Bot } from '@/lib/bbgo/manager'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { CreateStrategyDialog } from '@/components/user/CreateStrategyDialog'

export default function BotsPage() {
  const t = useTranslations('Bots')
  const userId = useUserId()
  const { mode: globalMode } = useTradingMode()
  const deleteStrategy = useDeleteStrategy()
  const [showCreate, setShowCreate] = useState(false)
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>
          <p className="text-sm text-muted-foreground">{t('modeBots', { mode: t(`mode.${globalMode}`) })}</p>
        </div>
        <Button onClick={() => setShowCreate(true)} className="rounded-full">
          <Plus className="mr-1.5 h-4 w-4" />
          {t('create')}
        </Button>
      </div>

      {userId && (
        <BotListView
          userId={userId}
          mode={globalMode}
          onDelete={(id) => setPendingDeleteId(id)}
          deleteDisabled={deleteStrategy.isPending}
        />
      )}

      {showCreate && userId && (
        <CreateStrategyDialog userId={userId} onClose={() => setShowCreate(false)} />
      )}

      {pendingDeleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" role="presentation" onClick={() => setPendingDeleteId(null)} onKeyDown={(e) => { if (e.key === 'Escape') setPendingDeleteId(null) }}>
          <div role="dialog" aria-modal="true" className="rounded-lg bg-card p-6 shadow-lg max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <p className="text-sm">{t('removeConfirm')}</p>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setPendingDeleteId(null)} className="rounded-md border px-4 py-2 text-sm hover:bg-muted">
                {t('cancel')}
              </button>
              <button
                onClick={() => {
                  deleteStrategy.mutate(
                    { userId, strategyId: pendingDeleteId },
                    {
                      onSuccess: () => setPendingDeleteId(null),
                      onError: (err) => { toast.error(err.message); setPendingDeleteId(null) },
                    },
                  )
                }}
                disabled={deleteStrategy.isPending}
                className="rounded-md bg-destructive px-4 py-2 text-sm text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
              >
                {t('remove')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function BotListView({ userId, mode, onDelete, deleteDisabled }: {
  userId: string
  mode: 'live' | 'paper'
  onDelete: (id: string) => void
  deleteDisabled: boolean
}) {
  const t = useTranslations('Bots')
  const { data: botsResp, isLoading, isError } = useBotList(userId, mode)
  const startInstance = useStartInstance()
  const stopInstance = useStopInstance()
  const [errorDialogBot, setErrorDialogBot] = useState<Bot | null>(null)

  if (isError) {
    return (
      <div className="rounded-lg border bg-destructive/10 p-6 text-center text-destructive">
        {t('errorLoading')}
      </div>
    )
  }

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

  const bots = botsResp?.bots ?? []

  if (bots.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
        {t('noStrategies')}
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {bots.map((bot) => {
        const status = bot.container_status
        const isRunning = status === 'running'
        const symbol = bot.symbol || (bot.config?.symbol as string) || ''
        const exchange = bot.exchange
        const title = bot.name?.trim() || bot.strategy
        const errorMsg = bot.last_error?.trim() || ''

        return (
          <div key={bot.id} className="flex items-center justify-between rounded-lg border bg-card p-4">
            <div className="flex items-center gap-3 min-w-0">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted shrink-0">
                <BotIcon className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <p className="font-medium truncate" title={title}>{title}</p>
                  <Badge variant="outline" className="rounded-full text-[10px] shrink-0">
                    {t(`mode.${bot.mode}`)}
                  </Badge>
                  <span className="text-[10px] font-mono text-muted-foreground/70 shrink-0">{bot.strategy}</span>
                </div>
                <p className="text-sm text-muted-foreground truncate">
                  {exchange}{symbol ? ` · ${symbol}` : ''}{bot.name?.trim() ? ` · ${bot.strategy}` : ''}
                </p>
                {status === 'error' && errorMsg && (
                  <button
                    type="button"
                    onClick={() => setErrorDialogBot(bot)}
                    className="mt-1 inline-flex items-center gap-1 rounded-md bg-destructive/10 px-2 py-0.5 text-xs text-destructive hover:bg-destructive/20"
                  >
                    <AlertCircle className="h-3 w-3 shrink-0" />
                    <span className="truncate max-w-[280px]">{errorMsg}</span>
                    <span className="shrink-0 underline-offset-2 hover:underline">{t('errorDetails')}</span>
                  </button>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
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
              {isRunning && (
                <Link
                  href={`/user/bots/${encodeURIComponent(bot.id)}?mode=${bot.mode}`}
                  className="rounded-md border px-3 py-1 text-sm hover:bg-muted"
                >
                  {t('dashboard')}
                </Link>
              )}
              {status === 'running' ? (
                <button
                  onClick={() => stopInstance.mutate(
                    { userId, instanceId: bot.id },
                    { onError: (err) => toast.error(err.message) },
                  )}
                  disabled={stopInstance.isPending}
                  aria-label={t('stop')}
                  className="rounded-md border px-3 py-1 text-sm hover:bg-muted disabled:opacity-50"
                >
                  <Square className="h-3 w-3" />
                </button>
              ) : (
                <button
                  onClick={() => startInstance.mutate(
                    { userId, instanceId: bot.id },
                    { onError: (err) => toast.error(err.message) },
                  )}
                  disabled={startInstance.isPending || status === 'starting'}
                  aria-label={t('start')}
                  className="rounded-md bg-primary px-3 py-1 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                >
                  <Play className="h-3 w-3" />
                </button>
              )}
              <button
                onClick={() => onDelete(bot.id)}
                disabled={deleteDisabled}
                aria-label={t('remove')}
                className="rounded-md border border-destructive px-2 py-1 text-destructive hover:bg-destructive hover:text-destructive-foreground disabled:opacity-50"
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          </div>
        )
      })}
      <Dialog open={!!errorDialogBot} onOpenChange={(open) => !open && setErrorDialogBot(null)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5" />
              {t('errorDialogTitle')}
            </DialogTitle>
            <DialogDescription>
              {t('errorDialogDescription', { name: errorDialogBot?.name?.trim() || errorDialogBot?.strategy || errorDialogBot?.id || '' })}
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="max-h-[60vh] rounded-md border bg-muted/30 p-3">
            <pre className="whitespace-pre-wrap break-all font-mono text-xs text-destructive">
              {errorDialogBot?.last_error}
            </pre>
          </ScrollArea>
        </DialogContent>
      </Dialog>
    </div>
  )
}
