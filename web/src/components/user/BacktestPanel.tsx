'use client'

import { useState, useCallback, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { useSubmitBacktest, useBacktestJob, useMarketSymbols, useMarketTicker } from '@/lib/bbgo/queries'
import { getStrategySchema, getStrategyDefaults, getStrategiesByCategory, ensureNumbers } from '@/lib/bbgo/strategies'
import { EXCHANGE_OPTIONS } from '@/lib/bbgo/constants'
import { StrategyConfigForm } from './StrategyConfigForm'
import { BacktestResultDisplay } from '@/components/backtest/BacktestEquityChart'

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

function formatParamValue(val: unknown): string {
  if (typeof val === 'boolean') return val ? '✓' : '✗'
  if (val === undefined || val === null || val === '') return ''
  return String(val)
}

export function BacktestPanel() {
  const t = useTranslations('Backtest')
  const ct = useTranslations('Categories')
  const submitBacktest = useSubmitBacktest()

  const [strategy, setStrategy] = useState('grid2')
  const [exchange, setExchange] = useState('binance')
  const [rawSymbol, setRawSymbol] = useState('BTCUSDT')
  const [config, setConfig] = useState<Record<string, unknown>>(getStrategyDefaults('grid2'))
  const [startTime, setStartTime] = useState(() => {
    const d = new Date()
    d.setMonth(d.getMonth() - 3)
    return toLocalDate(d)
  })
  const [endTime, setEndTime] = useState(() => toLocalDate(new Date()))
  const strategiesByCategory = getStrategiesByCategory({ excludeLiveOnly: true, excludeCrossExchange: true })
  const [activeJobId, setActiveJobId] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [downloading, setDownloading] = useState<string | null>(null)
  const [mode, setMode] = useState<'idle' | 'active'>('idle')

  const { data: symbolsData } = useMarketSymbols(exchange)

  const schema = getStrategySchema(strategy)
  const formFields = schema ? schema.fields.filter((f) => f.key !== 'symbol') : []

  const filteredSymbols = filterSymbols(symbolsData?.symbols ?? [])
  const symbol = filteredSymbols.includes(rawSymbol) ? rawSymbol : (filteredSymbols[0] ?? '')
  const displaySymbol = symbol

  const { data: tickerData } = useMarketTicker(exchange, symbol)

  const { data: activeJob } = useBacktestJob(activeJobId)
  const completedJob = activeJob && (activeJob.status === 'completed' || activeJob.status === 'failed') ? activeJob : null
  const lastResult = completedJob

  const isRunning = !!activeJobId && (activeJob?.status === 'pending' || activeJob?.status === 'downloading' || activeJob?.status === 'running')

  const handleSymbolChange = useCallback((newSymbol: string) => {
    setRawSymbol(newSymbol)
    setConfig((prev) => ({ ...prev, symbol: newSymbol }))
  }, [])

  useEffect(() => {
    if (!symbol || !tickerData?.ticker?.close) return
    if (!schema?.fields.some((f) => f.key === 'upperPrice' || f.key === 'lowerPrice')) return
    const price = tickerData.ticker.close
    const range = price * 0.2
    setConfig((prev) => ({
      ...prev,
      upperPrice: Math.round((price + range) * 100) / 100,
      lowerPrice: Math.round((price - range) * 100) / 100,
    }))
  }, [symbol, tickerData?.ticker?.close, schema])

  const handleStrategyChange = useCallback((newStrategy: string) => {
    setStrategy(newStrategy)
    setConfig(getStrategyDefaults(newStrategy))
  }, [])

  const handleDownload = async (file: string) => {
    if (!lastResult || downloading) return
    setDownloading(file)
    try {
      const res = await fetch(`/api/manager/backtest/jobs/${lastResult.id}/download?file=${file}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `backtest-${lastResult.id}-${file}.csv`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    } catch {
      setSubmitError(t('downloadFailed'))
    } finally {
      setDownloading(null)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitError(null)
    const numericConfig = ensureNumbers(schema, config)
    try {
      const result = await submitBacktest.mutateAsync({
        strategy,
        config: { ...numericConfig, exchange, symbol: displaySymbol },
        exchange,
        symbol: displaySymbol,
        start_time: startTime,
        end_time: endTime,
      })
      setActiveJobId(result.job_id)
      setMode('active')
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setSubmitError(message)
    }
  }

  const handleRerun = () => {
    setMode('idle')
    setActiveJobId(null)
    setSubmitError(null)
  }

  // -- Render: idle mode = full form --
  if (mode === 'idle' && !activeJobId) {
    return (
      <div className="space-y-6">
        <form onSubmit={handleSubmit} className="space-y-6 rounded-lg border bg-card p-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
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
                  setRawSymbol('')
                  setConfig(getStrategyDefaults(strategy))
                }}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
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
                onChange={(e) => handleSymbolChange(e.target.value)}
                disabled={!filteredSymbols.length}
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
                className="rounded border border-input px-2 py-0.5 text-xs hover:bg-muted/50"
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
              className="rounded border border-input px-2 py-0.5 text-xs hover:bg-muted/50"
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
              className="rounded-md bg-primary px-6 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              {t('run')}
            </button>
          </div>
        </form>

        {submitError && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
            <p className="text-sm text-destructive">{submitError}</p>
          </div>
        )}
      </div>
    )
  }

  // -- Render: active mode = summary + status/results --
  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <span className="text-sm font-semibold">{schema?.label ?? strategy}</span>
          <span className="text-xs text-muted-foreground">{exchange}</span>
          <span className="text-xs font-medium">{displaySymbol}</span>
          <span className="text-xs text-muted-foreground">{startTime} → {endTime}</span>
        </div>
        {formFields.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
            {formFields.map((f) => (
              <span key={f.key} className="text-xs text-muted-foreground">
                {f.key}: <span className="text-xs font-medium text-foreground">{formatParamValue(config[f.key])}</span>
              </span>
            ))}
          </div>
        )}
        <div className="mt-2 flex items-center gap-2">
          <button
            onClick={handleRerun}
            className="rounded border border-input px-3 py-1 text-xs hover:bg-muted/50"
          >
            {t('rerun')}
          </button>
          {lastResult?.status === 'completed' && (
            <>
              <button
                onClick={() => handleDownload('trades')}
                disabled={!!downloading}
                className="rounded border border-input px-3 py-1 text-xs hover:bg-muted/50 disabled:opacity-50"
              >
                {downloading === 'trades' ? '...' : t('downloadTrades')}
              </button>
              <button
                onClick={() => handleDownload('equity')}
                disabled={!!downloading}
                className="rounded border border-input px-3 py-1 text-xs hover:bg-muted/50 disabled:opacity-50"
              >
                {downloading === 'equity' ? '...' : t('downloadEquity')}
              </button>
            </>
          )}
        </div>
      </div>

      {submitError && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{submitError}</p>
        </div>
      )}

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

      {lastResult && lastResult.status === 'completed' && (
        <div className="rounded-lg border bg-card p-4">
          <BacktestResultDisplay
            output={lastResult.output}
            report={lastResult.report ?? null}
            equityCurveTSV={lastResult.equity_curve}
            jobId={lastResult.id}
            symbol={lastResult.symbol}
            exchange={lastResult.exchange}
            startTime={startTime}
            endTime={endTime}
          />
        </div>
      )}

      {lastResult && lastResult.status === 'failed' && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{lastResult.error || t('error')}</p>
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
