'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useCredentials, useCreateCredential, useDeleteCredential } from '@/lib/bbgo/queries'
import { EXCHANGES, EXCHANGES_REQUIRING_PASSPHRASE } from '@/lib/bbgo/constants'
import { useUserId } from '@/components/providers/user-id'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'


export function ApiKeyList() {
  const t = useTranslations('Settings.apiKeys')
  const bt = useTranslations('Bots')
  const userId = useUserId()
  const [showAdd, setShowAdd] = useState(false)
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null)
  const [exchange, setExchange] = useState('binance')
  const [apiKey, setApiKey] = useState('')
  const [apiSecret, setApiSecret] = useState('')
  const [passphrase, setPassphrase] = useState('')
  const [isTestnet, setIsTestnet] = useState(false)
  const [errors, setErrors] = useState<{ apiKey?: string; apiSecret?: string }>({})

  const { data: credentials = [], isLoading } = useCredentials(userId)
  const createMut = useCreateCredential()
  const deleteMut = useDeleteCredential()

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const newErrors: { apiKey?: string; apiSecret?: string } = {}
    if (!apiKey) newErrors.apiKey = t('required')
    if (!apiSecret) newErrors.apiSecret = t('required')
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors)
      return
    }
    setErrors({})
    if (!userId) {
      toast.error(t('authRequired'))
      return
    }

    createMut.mutate(
      { exchange, api_key: apiKey, api_secret: apiSecret, passphrase: passphrase || undefined, is_testnet: isTestnet },
      {
        onSuccess: (data) => {
          if (data.is_verified) {
            toast.success(isTestnet ? t('testnetAdded') : t('added'))
          } else {
            toast.error(t('verifyFailed'))
          }
          setShowAdd(false)
          setApiKey('')
          setApiSecret('')
          setPassphrase('')
          setIsTestnet(false)
        },
        onError: (err) => {
          toast.error(err.message)
        },
      },
    )
  }

  const handleDelete = (id: string) => {
    if (!userId) return
    deleteMut.mutate(
      { id, userId },
      { onError: (err) => toast.error(err.message) },
    )
  }

  return (
    <div className="space-y-4">
      {isLoading && (
        <div className="rounded-lg border bg-card p-6 text-center text-muted-foreground">
          {bt('loading')}
        </div>
      )}

      {!isLoading && credentials.length > 0 && (
        <div className="space-y-2">
          {credentials.map((cred) => (
            <div key={cred.id} className="flex items-center justify-between rounded-lg border bg-card px-4 py-3">
              <div className="flex items-center gap-3">
                <span className="rounded-md bg-muted px-2 py-1 text-xs font-medium uppercase">{cred.exchange}</span>
                {cred.is_testnet && (
                  <span className="rounded-md bg-yellow-100 text-yellow-800 px-2 py-1 text-xs">{bt('testnet')}</span>
                )}
                {cred.is_verified ? (
                  <span className="rounded-md bg-green-100 text-green-700 px-2 py-1 text-xs">{t('verified')}</span>
                ) : (
                  <span className="rounded-md bg-red-100 text-red-700 px-2 py-1 text-xs" title={cred.verify_error}>{t('verifyFailed')}</span>
                )}
              </div>
              <button
                onClick={() => setPendingDeleteId(cred.id)}
                disabled={deleteMut.isPending}
                aria-label={t('delete')}
                className="rounded-md border border-destructive px-3 py-1 text-sm text-destructive hover:bg-destructive hover:text-destructive-foreground disabled:opacity-50"
              >
                {t('delete')}
              </button>
            </div>
          ))}
        </div>
      )}

      {!isLoading && credentials.length === 0 && !showAdd && (
        <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
          {t('empty')}
        </div>
      )}

      {!showAdd && (
        <button
          onClick={() => setShowAdd(true)}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          {t('add')}
        </button>
      )}

      {showAdd && (
        <form onSubmit={handleSubmit} noValidate className="space-y-4 rounded-lg border bg-card p-6">
          <div>
            <label className="block text-sm font-medium mb-1">{t('exchange')}</label>
            <select
              value={exchange}
              onChange={(e) => setExchange(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {EXCHANGES.map((ex) => (
                <option key={ex} value={ex}>{ex}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('apiKey')}</label>
            <input
              type="text"
              required
              value={apiKey}
              onChange={(e) => { setApiKey(e.target.value); setErrors((p) => ({ ...p, apiKey: undefined })) }}
              className={cn('w-full rounded-md border border-input bg-background px-3 py-2 text-sm', errors.apiKey && 'border-destructive')}
            />
            {errors.apiKey && <p className="mt-1 text-xs text-destructive">{errors.apiKey}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('apiSecret')}</label>
            <input
              type="password"
              required
              value={apiSecret}
              onChange={(e) => { setApiSecret(e.target.value); setErrors((p) => ({ ...p, apiSecret: undefined })) }}
              className={cn('w-full rounded-md border border-input bg-background px-3 py-2 text-sm', errors.apiSecret && 'border-destructive')}
            />
            {errors.apiSecret && <p className="mt-1 text-xs text-destructive">{errors.apiSecret}</p>}
          </div>

          {EXCHANGES_REQUIRING_PASSPHRASE.includes(exchange) && (
            <div>
              <label className="block text-sm font-medium mb-1">{t('passphrase')}</label>
              <input
                type="password"
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
            </div>
          )}

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="testnet"
              checked={isTestnet}
              onChange={(e) => setIsTestnet(e.target.checked)}
              className="h-4 w-4 rounded border-input accent-primary"
            />
            <label htmlFor="testnet" className="text-sm">{t('testnet')}</label>
          </div>

          {createMut.isError && (
            <p className="text-sm text-destructive">{createMut.error.message}</p>
          )}

          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => setShowAdd(false)} className="rounded-md border px-4 py-2 text-sm">
              {bt('cancel')}
            </button>
            <button
              type="submit"
              disabled={createMut.isPending}
              className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {createMut.isPending ? bt('saving') : bt('save')}
            </button>
          </div>
        </form>
      )}
      {pendingDeleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setPendingDeleteId(null)}>
          <div className="rounded-lg bg-card p-6 shadow-lg max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <p className="text-sm">{t('delete')}?</p>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setPendingDeleteId(null)} className="rounded-md border px-4 py-2 text-sm hover:bg-muted">
                {bt('cancel')}
              </button>
              <button
                onClick={() => { handleDelete(pendingDeleteId); setPendingDeleteId(null) }}
                disabled={deleteMut.isPending}
                className="rounded-md bg-destructive px-4 py-2 text-sm text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
              >
                {t('delete')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
