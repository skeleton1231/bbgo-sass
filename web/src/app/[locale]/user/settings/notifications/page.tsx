'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import {
  fetchNotificationConfigs,
  createNotificationConfig,
  deleteNotificationConfig,
  testNotification,
  type NotificationConfigInfo,
} from '@/lib/bbgo/manager'
import { toast } from 'sonner'

export default function NotificationSettingsPage() {
  const t = useTranslations('Settings.notifications')
  const [configs, setConfigs] = useState<NotificationConfigInfo[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchNotificationConfigs()
      .then(setConfigs)
      .catch(() => toast.error(t('loadError')))
      .finally(() => setLoading(false))
  }, [t])

  async function handleDelete(id: string) {
    try {
      await deleteNotificationConfig(id)
      setConfigs((prev) => prev.filter((c) => c.id !== id))
      toast.success(t('deleted'))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('deleteError'))
    }
  }

  async function handleTest() {
    try {
      await testNotification()
      toast.success(t('testSent'))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('testError'))
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold">{t('title')}</h1>
        <p className="text-sm text-muted-foreground mt-1">{t('description')}</p>
      </div>

      <AddChannelForm
        type="telegram"
        onSubmit={async (data) => {
          const created = await createNotificationConfig(data)
          setConfigs((prev) => [...prev, created])
        }}
      />

      <AddChannelForm
        type="slack"
        onSubmit={async (data) => {
          const created = await createNotificationConfig(data)
          setConfigs((prev) => [...prev, created])
        }}
      />

      {configs.length > 0 && (
        <button onClick={handleTest} className="rounded-md border px-4 py-2 text-sm hover:bg-muted">
          {t('sendTest')}
        </button>
      )}

      <div className="space-y-3">
        {loading && <p className="text-sm text-muted-foreground">{t('loading')}</p>}
        {configs.map((cfg) => (
          <div key={cfg.id} className="flex items-center justify-between rounded-lg border p-4">
            <div>
              <p className="font-medium">{cfg.type === 'telegram' ? 'Telegram' : 'Slack'}</p>
              <div className="flex gap-3 mt-1">
                <span className={cn('text-xs rounded px-1.5 py-0.5', cfg.rules.trade_events ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500')}>
                  {t('trades')}
                </span>
                <span className={cn('text-xs rounded px-1.5 py-0.5', cfg.rules.order_events ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500')}>
                  {t('orders')}
                </span>
                <span className={cn('text-xs rounded px-1.5 py-0.5', cfg.rules.container_health ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500')}>
                  {t('health')}
                </span>
              </div>
            </div>
            <button onClick={() => handleDelete(cfg.id)} className="text-sm text-destructive hover:underline">
              {t('remove')}
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}

function AddChannelForm({
  type,
  onSubmit,
}: {
  type: 'telegram' | 'slack'
  onSubmit: (data: {
    type: 'telegram' | 'slack'
    token?: string
    chat_id?: string
    webhook_url?: string
    rules: { trade_events: boolean; order_events: boolean; container_health: boolean }
  }) => Promise<void>
}) {
  const t = useTranslations('Settings.notifications')
  const [open, setOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [rules, setRules] = useState({ trade_events: true, order_events: true, container_health: true })

  if (!open) {
    return (
      <button onClick={() => setOpen(true)} className="rounded-md border border-dashed px-4 py-3 text-sm text-muted-foreground hover:bg-muted w-full">
        + {type === 'telegram' ? t('addTelegram') : t('addSlack')}
      </button>
    )
  }

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSubmitting(true)
    const fd = new FormData(e.currentTarget)
    try {
      const data: Parameters<typeof onSubmit>[0] = { type, rules }
      if (type === 'telegram') {
        data.token = fd.get('token') as string
        data.chat_id = fd.get('chat_id') as string
      } else {
        data.webhook_url = fd.get('webhook_url') as string
      }
      await onSubmit(data)
      setOpen(false)
      toast.success(t('added'))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('addError'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-lg border p-4 space-y-4">
      <h3 className="font-medium">{type === 'telegram' ? 'Telegram' : 'Slack'}</h3>
      {type === 'telegram' ? (
        <div className="space-y-3">
          <input name="token" placeholder={t('botToken')} required className="w-full rounded-md border px-3 py-2 text-sm" />
          <input name="chat_id" placeholder={t('chatId')} required className="w-full rounded-md border px-3 py-2 text-sm" />
        </div>
      ) : (
        <input name="webhook_url" placeholder={t('webhookUrl')} required className="w-full rounded-md border px-3 py-2 text-sm" />
      )}
      <div className="flex gap-4 text-sm">
        <label className="flex items-center gap-1">
          <input type="checkbox" checked={rules.trade_events} onChange={(e) => setRules({ ...rules, trade_events: e.target.checked })} />
          {t('trades')}
        </label>
        <label className="flex items-center gap-1">
          <input type="checkbox" checked={rules.order_events} onChange={(e) => setRules({ ...rules, order_events: e.target.checked })} />
          {t('orders')}
        </label>
        <label className="flex items-center gap-1">
          <input type="checkbox" checked={rules.container_health} onChange={(e) => setRules({ ...rules, container_health: e.target.checked })} />
          {t('health')}
        </label>
      </div>
      <div className="flex gap-2">
        <button type="submit" disabled={submitting} className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50">
          {submitting ? t('adding') : t('add')}
        </button>
        <button type="button" onClick={() => setOpen(false)} className="rounded-md border px-4 py-2 text-sm hover:bg-muted">
          {t('cancel')}
        </button>
      </div>
    </form>
  )
}
