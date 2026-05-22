'use client'

import { useState, useEffect, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import { useRunBacktest } from '@/lib/bbgo/queries'
import { getStrategySchema, getStrategyDefaults, getAllStrategies } from '@/lib/bbgo/strategies'
import { StrategyConfigForm } from './StrategyConfigForm'

interface BacktestReport {
  id: string
  strategy: string
  total_profit: string
  win_rate: string
  total_trades: number
  start_date: string
  end_date: string
  created_at: string
}

export function BacktestPanel({ userId }: { userId: string }) {
  const t = useTranslations('Backtest')
  const runBacktest = useRunBacktest()
  const strategies = getAllStrategies()

  const [strategy, setStrategy] = useState('grid2')
  const [config, setConfig] = useState<Record<string, unknown>>(getStrategyDefaults('grid2'))
  const [history, setHistory] = useState<BacktestReport[]>([])

  const schema = getStrategySchema(strategy)

  useEffect(() => {
    loadHistory()
  }, [userId])

  const loadHistory = async () => {
    if (!userId) return
    const { createClient } = await import('@/lib/supabase/client')
    const supabase = createClient()
    const { data } = await supabase
      .from('backtest_reports')
      .select('id, strategy, total_profit, win_rate, total_trades, start_date, end_date, created_at')
      .eq('user_id', userId)
      .order('created_at', { ascending: false })
      .limit(20)
    if (data) setHistory(data)
  }

  const handleStrategyChange = useCallback((newStrategy: string) => {
    setStrategy(newStrategy)
    setConfig(getStrategyDefaults(newStrategy))
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await runBacktest.mutateAsync({
      strategy,
      config,
    })
    loadHistory()
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleSubmit} className="space-y-4 rounded-lg border bg-card p-6">
        <div className="grid gap-4 md:grid-cols-2">
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
        </div>

        {schema && (
          <StrategyConfigForm
            fields={schema.fields}
            values={config}
            onChange={setConfig}
          />
        )}

        <button
          type="submit"
          disabled={runBacktest.isPending}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {runBacktest.isPending ? 'Running...' : t('run')}
        </button>
      </form>

      {runBacktest.data && (
        <div className="rounded-lg border bg-card p-4">
          <h3 className="text-sm font-semibold mb-2">Backtest Output</h3>
          <pre className="whitespace-pre-wrap text-xs text-muted-foreground max-h-96 overflow-y-auto">
            {runBacktest.data.output}
          </pre>
        </div>
      )}

      {runBacktest.isError && (
        <p className="text-sm text-destructive">{(runBacktest.error as Error).message}</p>
      )}

      {history.length > 0 && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">{t('results')}</h2>
          </div>
          <div className="divide-y">
            {history.map((report) => (
              <div key={report.id} className="flex items-center justify-between px-4 py-3">
                <div>
                  <p className="text-sm font-medium">{report.strategy}</p>
                  <p className="text-xs text-muted-foreground">
                    {report.start_date} ~ {report.end_date}
                  </p>
                </div>
                <div className="flex items-center gap-4 text-sm">
                  <span>Profit: <strong>{report.total_profit}</strong></span>
                  <span>Win: <strong>{report.win_rate}</strong></span>
                  <span>Trades: {report.total_trades}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
