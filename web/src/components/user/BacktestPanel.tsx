'use client'

import { useState, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import { useRunBacktest } from '@/lib/bbgo/queries'
import { getStrategySchema, getStrategyDefaults, getStrategiesByCategory } from '@/lib/bbgo/strategies'
import { CATEGORY_LABELS, EXCHANGE_OPTIONS } from '@/lib/bbgo/constants'
import { StrategyConfigForm } from './StrategyConfigForm'

export function BacktestPanel({ userId }: { userId: string }) {
  const t = useTranslations('Backtest')
  const runBacktest = useRunBacktest()
  const strategiesByCategory = getStrategiesByCategory({ excludeLiveOnly: true })

  const [strategy, setStrategy] = useState('grid2')
  const [exchange, setExchange] = useState('binance')
  const [config, setConfig] = useState<Record<string, unknown>>(getStrategyDefaults('grid2'))
  const [startTime, setStartTime] = useState('2024-01-01')
  const [endTime, setEndTime] = useState('2024-03-01')

  const schema = getStrategySchema(strategy)

  const handleStrategyChange = useCallback((newStrategy: string) => {
    setStrategy(newStrategy)
    setConfig(getStrategyDefaults(newStrategy))
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await runBacktest.mutateAsync({
        strategy,
        config: { ...config, exchange },
        start_time: startTime,
        end_time: endTime,
      })
    } catch {
      // Error is available via runBacktest.error
    }
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleSubmit} className="space-y-6 rounded-lg border bg-card p-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('strategy')}</label>
            <select
              value={strategy}
              onChange={(e) => handleStrategyChange(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {Object.entries(strategiesByCategory).map(([cat, items]) => (
                <optgroup key={cat} label={CATEGORY_LABELS[cat] || cat}>
                  {items.map((s) => (
                    <option key={s.id} value={s.id}>{s.label}</option>
                  ))}
                </optgroup>
              ))}
            </select>
            {schema && (
              <p className="mt-1 text-xs text-muted-foreground">{schema.description}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('exchange')}</label>
            <select
              value={exchange}
              onChange={(e) => setExchange(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {EXCHANGE_OPTIONS.map((ex) => (
                <option key={ex.id} value={ex.id}>{ex.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('startDate')}</label>
            <input
              type="date"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('endDate')}</label>
            <input
              type="date"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          </div>
        </div>

        {schema && schema.fields.length > 0 && (
          <div className="border-t pt-4">
            <StrategyConfigForm
              fields={schema.fields}
              values={config}
              onChange={setConfig}
            />
          </div>
        )}

        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={runBacktest.isPending}
            className="rounded-md bg-primary px-6 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {runBacktest.isPending ? t('running') : t('run')}
          </button>
          {runBacktest.isPending && (
            <span className="text-xs text-muted-foreground">{t('backtestDuration')}</span>
          )}
        </div>
      </form>

      {runBacktest.isError && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{runBacktest.error instanceof Error ? runBacktest.error.message : t('error')}</p>
        </div>
      )}

      {runBacktest.data && (
        <div className="rounded-lg border bg-card p-4">
          <h3 className="text-sm font-semibold mb-2">{t('backtestOutput')}</h3>
          <pre className="whitespace-pre-wrap text-xs text-muted-foreground max-h-[500px] overflow-y-auto rounded bg-muted/50 p-3">
            {runBacktest.data.output.replace(/\x1b\[[0-9;]*m/g, '')}
          </pre>
        </div>
      )}
    </div>
  )
}
