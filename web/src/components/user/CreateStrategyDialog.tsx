'use client'

import { useState, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import { useCreateStrategy, useCredentials } from '@/lib/bbgo/queries'
import { useStrategyRegistry } from '@/components/providers/strategy-registry'
import { getStrategySchema, getStrategyDefaults, getStrategiesByCategory, ensureTypes, nestConfig, type SessionRole } from '@/lib/bbgo/strategies'
import { EXCHANGES } from '@/lib/bbgo/constants'
import { useTradingMode } from '@/components/providers/trading-mode'
import { StrategyConfigForm } from './StrategyConfigForm'
import { toast } from 'sonner'

const ENV_PREFIXES: Record<string, string> = {
  binance: 'BINANCE',
  okex: 'OKEX',
  bybit: 'BYBIT',
  bitget: 'BITGET',
  kucoin: 'KUCOIN',
  max: 'MAX',
  coinbase: 'COINBASE',
  bitfinex: 'BITFINEX',
}

export function CreateStrategyDialog({ userId, onClose }: { userId: string; onClose: () => void }) {
  const t = useTranslations('Bots')
  const ct = useTranslations('Categories')
  const createStrategy = useCreateStrategy()
  const { data: credentials } = useCredentials(userId)
  const registry = useStrategyRegistry()
  const hasExchangeCreds = (ex: string, testnet = false) =>
    (credentials ?? []).some(c => c.exchange === ex && c.is_testnet === testnet)

  const [name, setName] = useState('')
  const [exchange, setExchange] = useState('binance')
  const [strategy, setStrategy] = useState('grid2')
  const { mode: globalMode } = useTradingMode()
  const [mode, setMode] = useState<'live' | 'paper'>(globalMode)
  const [config, setConfig] = useState<Record<string, unknown>>(getStrategyDefaults('grid2', registry))
  const [sessionExchanges, setSessionExchanges] = useState<Record<string, string>>({})

  const handleStrategyChange = useCallback((newStrategy: string) => {
    setStrategy(newStrategy)
    setConfig(getStrategyDefaults(newStrategy, registry))
    if (getStrategySchema(newStrategy, registry)?.liveOnly) {
      setMode('live')
    }
    const s = getStrategySchema(newStrategy, registry)
    if (s?.sessionRoles) {
      const defaults: Record<string, string> = {}
      for (const role of s.sessionRoles) {
        defaults[role.name] = 'binance'
      }
      setSessionExchanges(defaults)
    } else {
      setSessionExchanges({})
    }
  }, [registry])

  const effectiveExchange = exchange

  const strategiesByCategory = getStrategiesByCategory(undefined, registry)
  const schema = getStrategySchema(strategy, registry)
  const isCrossExchange = schema?.crossExchange === true
  const isLiveOnly = schema?.liveOnly === true

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const numericConfig = nestConfig(ensureTypes(schema, config))
    const onError = (err: Error) => {
      toast.error(err.message)
    }
    if (isCrossExchange && schema?.sessionRoles) {
      const sessions = schema.sessionRoles.map((role: SessionRole) => {
        const ex = mode === 'paper' ? 'binance' : (sessionExchanges[role.name] || 'binance')
        return {
          name: role.name,
          exchange: ex,
          envVarPrefix: ENV_PREFIXES[ex] || ex.toUpperCase(),
          futures: role.futures,
        }
      })
      createStrategy.mutate(
        {
          userId,
          name,
          exchange: '',
          strategy,
          config: numericConfig,
          mode,
          crossExchange: true,
          sessions,
        },
        { onSuccess: onClose, onError }
      )
    } else {
      createStrategy.mutate(
        {
          userId,
          name,
          exchange: effectiveExchange,
          strategy,
          config: numericConfig,
          mode,
        },
        { onSuccess: onClose, onError }
      )
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" role="presentation" onClick={onClose} onKeyDown={(e) => { if (e.key === 'Escape') onClose() }}>
      <form
        role="dialog"
        aria-modal="true"

        onClick={(e) => e.stopPropagation()}
        onSubmit={handleSubmit}
        className="w-full max-w-lg max-h-[90vh] overflow-y-auto space-y-4 rounded-lg bg-card p-6 shadow-lg"
      >
        <h2 className="text-lg font-semibold">{t('create')}</h2>

        <div>
          <label className="block text-sm font-medium mb-1">{t('name')}</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('exchange')}</label>
            <select
              value={exchange}
              onChange={(e) => setExchange(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              disabled={isCrossExchange}
            >
              {EXCHANGES.map((ex) => (
                <option key={ex} value={ex}>{ex}</option>
              ))}
            </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">{t('strategy')}</label>
          <select
            value={strategy}
            onChange={(e) => handleStrategyChange(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          >
            {Object.entries(strategiesByCategory).map(([cat, items]) => (
              <optgroup key={cat} label={ct(cat)}>
                {items.map((s) => (
                  <option key={s.id} value={s.id}>{s.label}</option>
                ))}
              </optgroup>
            ))}
          </select>
        </div>

        {isCrossExchange && schema?.sessionRoles && (
          <div className="space-y-3 rounded-md border border-border p-3">
            <p className="text-sm font-medium">{t('sessionRoles')}</p>
            {schema.sessionRoles.map((role) => (
              <div key={role.name} className="flex items-center gap-3">
                <span className="text-sm text-muted-foreground w-32">{role.label}</span>
                <select
                  value={sessionExchanges[role.name] || 'binance'}
                  onChange={(e) => setSessionExchanges((prev) => ({ ...prev, [role.name]: e.target.value }))}
                  className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
                >
                  {EXCHANGES.map((ex) => (
                    <option key={ex} value={ex}>{ex}{role.futures ? ` (${t('futures')})` : ''}</option>
                  ))}
                </select>
                {role.futures && (
                  <span className="text-xs text-muted-foreground">{t('futures')}</span>
                )}
              </div>
            ))}
          </div>
        )}

        <div>
          <label className="block text-sm font-medium mb-1">{t('modeLabel')}</label>
          <div className="flex gap-4">
            {(['live', 'paper'] as const).map((m) => (
              <label key={m} className={`flex items-center gap-2 text-sm${isLiveOnly && m === 'paper' ? ' opacity-50' : ''}`}>
                <input
                  type="radio"
                  name="mode"
                  value={m}
                  checked={mode === m}
                  onChange={() => setMode(m)}
                  disabled={isLiveOnly && m === 'paper'}
                  className="h-4 w-4 border-input accent-primary"
                />
                {t(`mode.${m}`)}
              </label>
            ))}
          </div>
          {isLiveOnly && globalMode === 'paper' && (
            <p className="mt-1 text-xs text-yellow-600 bg-yellow-50 rounded px-2 py-1">
              {t('liveOnlyOverrideHint')}
            </p>
          )}
          {mode === 'live' && !isCrossExchange && !hasExchangeCreds(exchange, false) && (
            <p className="mt-1 text-xs text-destructive">
              {t('noCredsForExchange', { exchange })}
            </p>
          )}
          {mode === 'live' && isCrossExchange && schema?.sessionRoles && (
            <div className="mt-1 space-y-0.5">
              {schema.sessionRoles.filter(r => !hasExchangeCreds(sessionExchanges[r.name] || 'binance', false)).map(r => (
                <p key={r.name} className="text-xs text-destructive">
                  {t('noCredsForRole', { role: r.label, exchange: sessionExchanges[r.name] || 'binance' })}
                </p>
              ))}
            </div>
          )}
        </div>

        {schema && (
          <StrategyConfigForm
            fields={schema.fields}
            values={config}
            onChange={setConfig}
          />
        )}

        {createStrategy.isError && (
          <p className="text-sm text-destructive">
            {t('createError')}
          </p>
        )}

        <div className="flex justify-end gap-2 pt-2">
          <button type="button" onClick={onClose} className="rounded-md border px-4 py-2 text-sm">
            {t('cancel')}
          </button>
          <button
            type="submit"
            disabled={createStrategy.isPending}
            className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {createStrategy.isPending ? t('creating') : t('create')}
          </button>
        </div>
      </form>
    </div>
  )
}
