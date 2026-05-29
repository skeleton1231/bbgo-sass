'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { useCredentials, useCreateCredential, useDeleteCredential } from '@/lib/bbgo/queries'
import { EXCHANGES, EXCHANGES_REQUIRING_PASSPHRASE } from '@/lib/bbgo/constants'
import { cn } from '@/lib/utils'


export function ApiKeyList() {
  const t = useTranslations('Settings.apiKeys')
  const bt = useTranslations('Bots')
  const [userId, setUserId] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const [exchange, setExchange] = useState('binance')
  const [apiKey, setApiKey] = useState('')
  const [apiSecret, setApiSecret] = useState('')
  const [passphrase, setPassphrase] = useState('')
  const [isTestnet, setIsTestnet] = useState(false)
  const [errors, setErrors] = useState<{ apiKey?: string; apiSecret?: string }>({})

  useEffect(() => {
    const loadUser = async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data } = await supabase.auth.getUser()
      if (data.user) setUserId(data.user.id)
    }
    loadUser()
  }, [])

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
    if (!userId) return

    createMut.mutate(
      { exchange, api_key: apiKey, api_secret: apiSecret, passphrase: passphrase || undefined, is_testnet: isTestnet },
      {
        onSuccess: () => {
          setShowAdd(false)
          setApiKey('')
          setApiSecret('')
          setPassphrase('')
          setIsTestnet(false)
        },
      },
    )
  }

  const handleDelete = (id: string) => {
    if (!userId) return
    deleteMut.mutate({ id, userId })
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
                  <span className="rounded-md bg-gray-100 text-gray-500 px-2 py-1 text-xs">{t('verificationFailed')}</span>
                )}
              </div>
              <button
                onClick={() => handleDelete(cred.id)}
                disabled={deleteMut.isPending}
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
    </div>
  )
}
