'use client'

import { useState, useCallback, useEffect, useRef } from 'react'
import { useTranslations } from 'next-intl'
import { useSubmitBacktest, useBacktestJob, useBacktestJobs, useMarketSymbols, useMarketTicker } from '@/lib/bbgo/queries'
import type { BacktestJob } from '@/lib/bbgo/queries'
import { getStrategySchema, getStrategyDefaults, getStrategiesByCategory } from '@/lib/bbgo/strategies'
import { EXCHANGE_OPTIONS } from '@/lib/bbgo/constants'
import { StrategyConfigForm } from './StrategyConfigForm'
import { BacktestEquityChart } from '@/components/backtest/BacktestEquityChart'

const QUOTE_CURRENCIES = ['USDT', 'BUSD', 'USDC', 'TUSD', 'FDUSD', 'BTC', 'ETH', 'BNB']

function isValidTradingPair(symbol: string): boolean {
  return QUOTE_CURRENCIES.some((q) => symbol.endsWith(q) && symbol.length > q.length)
}

function filterSymbols(symbols: string[]): string[] {
  return symbols.filter(isValidTradingPair)
}

function toLocalDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function BacktestPanel({ userId }: { userId: string }) {
  const t = useTranslations('Backtest')
  const ct = useTranslations('Categories')
  const submitBacktest = useSubmitBacktest()
  const { data: jobsData } = useBacktestJobs()

  const [strategy, setStrategy] = useState('grid2')
  const [exchange, setExchange] = useState('binance')
  const [symbol, setSymbol] = useState('BTCUSDT')
  const [config, setConfig] = useState<Record<string, unknown>>(getStrategyDefaults('grid2'))
  const [startTime, setStartTime] = useState('2024-01-01')
  const [endTime, setEndTime] = useState('2024-03-01')
  const strategiesByCategory = getStrategiesByCategory({ excludeLiveOnly: true, excludeCrossExchange: true })
  const [activeJobId, setActiveJobId] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<BacktestJob | null>(null)
  const prevSymbolRef = useRef(symbol)

  const { data: symbolsData } = useMarketSymbols(exchange)
  const { data: tickerData } = useMarketTicker(exchange, symbol)

  const schema = getStrategySchema(strategy)
  const formFields = schema ? schema.fields.filter((f) => f.key !== 'symbol') : []

  const filteredSymbols = filterSymbols(symbolsData?.symbols ?? [])
  const displaySymbol = filteredSymbols.includes(symbol) ? symbol : (filteredSymbols[0] ?? '')

  useEffect(() => {
    if (displaySymbol && displaySymbol !== symbol) {
      setSymbol(displaySymbol)
      setConfig((prev) => ({ ...prev, symbol: displaySymbol }))
    }
  }, [displaySymbol, symbol])

  const { data: activeJob } = useBacktestJob(activeJobId)

  // Auto-set upperPrice/lowerPrice from ticker only when symbol changes
  useEffect(() => {
    if (prevSymbolRef.current === symbol) return
    prevSymbolRef.current = symbol
    const ticker = tickerData?.ticker
    if (!ticker || !ticker.close) return
    const hasPriceFields = schema?.fields.some((f) => f.key === 'upperPrice' || f.key === 'lowerPrice')
    if (!hasPriceFields) return
    const price = ticker.close
    const range = price * 0.2
    setConfig((prev) => ({
      ...prev,
      symbol,
      upperPrice: Math.round((price + range) * 100) / 100,
      lowerPrice: Math.round((price - range) * 100) / 100,
    }))
  }, [tickerData, schema?.fields, symbol])

  useEffect(() => {
    if (activeJob && (activeJob.status === 'completed' || activeJob.status === 'failed')) {
      setLastResult(activeJob)
    }
  }, [activeJob])

  const handleStrategyChange = useCallback((newStrategy: string) => {
    setStrategy(newStrategy)
    setConfig(getStrategyDefaults(newStrategy))
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLastResult(null)
    try {
      const result = await submitBacktest.mutateAsync({
        strategy,
        config: { ...config, exchange, symbol: displaySymbol },
        exchange,
        symbol: displaySymbol,
        start_time: startTime,
        end_time: endTime,
      })
      setActiveJobId(result.job_id)
    } catch {
      // Error is displayed via submitBacktest.isError below
    }
  }

  const isRunning = !!activeJobId && (activeJob?.status === 'pending' || activeJob?.status === 'downloading' || activeJob?.status === 'running')

  const recentJobs = (jobsData?.jobs ?? [])
    .filter((j) => j.status === 'completed' || j.status === 'failed')
    .slice(0, 5)

  return (
    <div className="space-y-6">
      <form onSubmit={handleSubmit} className="space-y-6 rounded-lg border bg-card p-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
          <div>
            <label className="block text-sm font-medium mb-1">{t('strategy')}</label>
            <select
              value={strategy}
              onChange={(e) => handleStrategyChange(e.target.value)}
              disabled={isRunning}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-50"
            >
              {Object.entries(strategiesByCategory).map(([cat, items]) => (
                <optgroup key={cat} label={ct(cat)}>
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
              onChange={(e) => {
                setExchange(e.target.value)
                setSymbol('')
                setConfig(getStrategyDefaults(strategy))
              }}
              disabled={isRunning}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-50"
            >
              {EXCHANGE_OPTIONS.map((ex) => (
                <option key={ex.id} value={ex.id}>{ex.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('symbol')}</label>
            <select
              value={displaySymbol}
              onChange={(e) => {
                const newSymbol = e.target.value
                setSymbol(newSymbol)
                setConfig((prev) => ({
                  ...prev,
                  symbol: newSymbol,
                }))
              }}
              disabled={isRunning || !filteredSymbols.length}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-50"
            >
              {filteredSymbols.length ? (
                filteredSymbols.map((s) => (
                  <option key={s} value={s}>{s}</option>
                ))
              ) : (
                <option value="">{t('loading')}</option>
              )}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('startDate')}</label>
            <input
              type="date"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              disabled={isRunning}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-50"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">{t('endDate')}</label>
            <input
              type="date"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              disabled={isRunning}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm disabled:opacity-50"
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{t('quickRange')}</span>
          {([
            { label: '1M', months: 1 },
            { label: '3M', months: 3 },
            { label: '6M', months: 6 },
            { label: '1Y', months: 12 },
          ] as const).map(({ label, months }) => (
            <button
              key={label}
              type="button"
              onClick={() => {
                const end = new Date()
                const start = new Date()
                start.setMonth(start.getMonth() - months)
                setEndTime(toLocalDate(end))
                setStartTime(toLocalDate(start))
              }}
              disabled={isRunning}
              className="rounded border border-input px-2 py-0.5 text-xs hover:bg-muted/50 disabled:opacity-50"
            >
              {label}
            </button>
          ))}
          <button
            type="button"
            onClick={() => {
              const now = new Date()
              const ytd = new Date(now.getFullYear(), 0, 1)
              setEndTime(toLocalDate(now))
              setStartTime(toLocalDate(ytd))
            }}
            disabled={isRunning}
            className="rounded border border-input px-2 py-0.5 text-xs hover:bg-muted/50 disabled:opacity-50"
          >
            YTD
          </button>
        </div>

        {formFields.length > 0 && (
          <div className="border-t pt-4">
            <StrategyConfigForm
              fields={formFields}
              values={config}
              onChange={setConfig}
            />
          </div>
        )}

        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={isRunning}
            className="rounded-md bg-primary px-6 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {isRunning ? t('running') : t('run')}
          </button>
        </div>
      </form>

      {isRunning && activeJob && (
        <div className="rounded-lg border bg-card p-4">
          <div className="flex items-center gap-3">
            <StatusBadge status={activeJob.status} />
            <span className="text-sm text-muted-foreground">
              {activeJob.status === 'downloading' && t('downloadingData')}
              {activeJob.status === 'running' && t('runningBacktest')}
              {activeJob.status === 'pending' && t('waitingToStart')}
            </span>
          </div>
          {activeJob.progress && (
            <p className="mt-2 text-xs text-muted-foreground">{activeJob.progress}</p>
          )}
        </div>
      )}

      {submitBacktest.isError && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{submitBacktest.error instanceof Error ? submitBacktest.error.message : t('error')}</p>
        </div>
      )}

      {lastResult && lastResult.status === 'completed' && lastResult.output && (
        <div className="rounded-lg border bg-card p-4 space-y-4">
          <BacktestEquityChart output={lastResult.output} />
          <details>
            <summary className="cursor-pointer text-sm font-medium text-muted-foreground hover:text-foreground">{t('backtestOutput')}</summary>
            <pre className="mt-2 whitespace-pre-wrap text-xs text-muted-foreground max-h-[500px] overflow-y-auto rounded bg-muted/50 p-3">
              {lastResult.output.replace(/\x1b\[[0-9;]*m/g, '')}
            </pre>
          </details>
        </div>
      )}

      {lastResult && lastResult.status === 'failed' && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{lastResult.error || t('error')}</p>
        </div>
      )}

      {recentJobs.length > 0 && (
        <div className="rounded-lg border bg-card p-4">
          <h3 className="text-sm font-semibold mb-3">{t('recentJobs')}</h3>
          <div className="space-y-2">
            {recentJobs.map((job) => (
              <button
                key={job.id}
                onClick={() => setLastResult(job)}
                className="w-full text-left flex items-center justify-between rounded-md px-3 py-2 text-sm hover:bg-muted/50 transition-colors"
              >
                <span className="truncate">
                  {job.strategy} / {job.symbol} / {job.start_time}
                </span>
                <StatusBadge status={job.status} />
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

const STATUS_LABEL_KEYS: Record<string, string> = {
  pending: 'statusPending',
  downloading: 'statusDownloading',
  running: 'statusRunning',
  completed: 'statusCompleted',
  failed: 'statusFailed',
}

const STATUS_COLORS: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
  downloading: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  running: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  completed: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
}

function StatusBadge({ status }: { status: string }) {
  const t = useTranslations('Backtest')
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[status] || 'bg-gray-100 text-gray-800'}`}>
      {t(STATUS_LABEL_KEYS[status] ?? status)}
    </span>
  )
}
