'use client'

import { useState, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import { useCreateStrategy } from '@/lib/bbgo/queries'
import { getStrategySchema, getStrategyDefaults, getAllStrategies } from '@/lib/bbgo/strategies'
import { EXCHANGES } from '@/lib/bbgo/constants'
import { StrategyConfigForm } from './StrategyConfigForm'


export function CreateStrategyDialog({ userId, onClose }: { userId: string; onClose: () => void }) {
  const t = useTranslations('Bots')
  const createStrategy = useCreateStrategy()

  const [name, setName] = useState('')
  const [exchange, setExchange] = useState('binance')
  const [strategy, setStrategy] = useState('grid2')
  const [mode, setMode] = useState<'live' | 'paper'>('paper')
  const [config, setConfig] = useState<Record<string, unknown>>(getStrategyDefaults('grid2'))

  const handleStrategyChange = useCallback((newStrategy: string) => {
    setStrategy(newStrategy)
    setConfig(getStrategyDefaults(newStrategy))
  }, [])

  const strategies = getAllStrategies()
  const schema = getStrategySchema(strategy)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createStrategy.mutate(
      {
        userId,
        name,
        exchange,
        strategy,
        config,
        mode,
      },
      { onSuccess: onClose }
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-lg max-h-[90vh] overflow-y-auto space-y-4 rounded-lg bg-card p-6 shadow-lg"
      >
        <h2 className="text-lg font-semibold">{t('create')}</h2>

        <div>
          <label className="block text-sm font-medium mb-1">Name</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Exchange</label>
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
          <label className="block text-sm font-medium mb-1">Strategy</label>
          <select
            value={strategy}
            onChange={(e) => handleStrategyChange(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          >
            {strategies.map((s) => (
              <option key={s.id} value={s.id}>{s.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Mode</label>
          <div className="flex gap-4">
            {(['live', 'paper'] as const).map((m) => (
              <label key={m} className="flex items-center gap-2 text-sm">
                <input
                  type="radio"
                  name="mode"
                  value={m}
                  checked={mode === m}
                  onChange={() => setMode(m)}
                />
                {t(`mode.${m}`)}
              </label>
            ))}
          </div>
        </div>

        {schema && (
          <StrategyConfigForm
            fields={schema.fields}
            values={config}
            onChange={setConfig}
          />
        )}

        <div className="flex justify-end gap-2 pt-2">
          <button type="button" onClick={onClose} className="rounded-md border px-4 py-2 text-sm">
            Cancel
          </button>
          <button
            type="submit"
            disabled={createStrategy.isPending}
            className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {createStrategy.isPending ? 'Creating...' : t('create')}
          </button>
        </div>
      </form>
    </div>
  )
}
