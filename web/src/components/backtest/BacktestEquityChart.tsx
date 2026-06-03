'use client'

import { useMemo, useState, useEffect, useRef, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  ReferenceLine,
} from 'recharts'
import {
  createChart,
  createSeriesMarkers,
  CandlestickSeries,
  HistogramSeries,
  ColorType,
  type Time,
  type IChartApi,
  type ISeriesApi,
  type SeriesType,
  type DeepPartial,
  type ChartOptions,
  type ISeriesMarkersPluginApi,
} from 'lightweight-charts'
import type { KlineCandle, TradeMarker } from '@/components/chart/CandlestickChart'
import { fetchMarketKlines } from '@/lib/bbgo/manager'
import type { BacktestReport, BacktestSymbolReport } from '@/lib/bbgo/queries'

interface EquityPoint {
  time: string
  value: number
}

interface ParsedBacktest {
  equityCurve: EquityPoint[]
  symbolReport: BacktestSymbolReport | null
  report: BacktestReport | null
}

function parseEquityCurveTSV(tsv: string): EquityPoint[] {
  const lines = tsv.trim().split('\n')
  if (lines.length < 2) return []
  const points: EquityPoint[] = []
  for (let i = 1; i < lines.length; i++) {
    const parts = lines[i]!.split('\t')
    if (parts.length >= 2 && parts[0] && parts[1]) {
      const val = parseFloat(parts[1])
      if (!isNaN(val)) {
        points.push({ time: parts[0], value: val })
      }
    }
  }
  return points
}

function parseLegacyOutput(output: string): ParsedBacktest {
  const cleanOutput = output.replace(/\x1b\[[0-9;]*m/g, '')
  const TRADE_PROFIT_PATTERN = /trade\s+#?(\d+).*?profit[:\s]+([-+]?[\d.,]+)/gi
  const equityCurve: EquityPoint[] = []
  let cumulative = 0
  let match: RegExpExecArray | null
  const regex = new RegExp(TRADE_PROFIT_PATTERN.source, 'gi')
  let tradeNum = 0

  while ((match = regex.exec(cleanOutput)) !== null) {
    tradeNum++
    const profit = parseFloat(match[2]!.replace(/,/g, ''))
    cumulative += profit
    equityCurve.push({ time: String(tradeNum), value: Math.round(cumulative * 100) / 100 })
  }

  return { equityCurve, symbolReport: null, report: null }
}

function parseKlineTSV(tsv: string): KlineCandle[] {
  const lines = tsv.trim().split('\n')
  if (lines.length < 2) return []
  const header = lines[0]!.split('\t').map((h) => h.trim().toLowerCase())
  const startIdx = header.indexOf('starttime')
  const openIdx = header.indexOf('open')
  const highIdx = header.indexOf('high')
  const lowIdx = header.indexOf('low')
  const closeIdx = header.indexOf('close')
  const volIdx = header.indexOf('volume')
  if (startIdx === -1 || openIdx === -1 || highIdx === -1 || lowIdx === -1 || closeIdx === -1) return []

  const maxIdx = Math.max(startIdx, openIdx, highIdx, lowIdx, closeIdx)
  const candles: KlineCandle[] = []
  for (let i = 1; i < lines.length; i++) {
    const parts = lines[i]!.split('\t')
    if (parts.length <= maxIdx) continue
    const time = Number(parts[startIdx])
    const open = Number(parts[openIdx])
    const high = Number(parts[highIdx])
    const low = Number(parts[lowIdx])
    const close = Number(parts[closeIdx])
    const volume = volIdx !== -1 ? Number(parts[volIdx]) : 0
    if (isNaN(time) || isNaN(open) || isNaN(high) || isNaN(low) || isNaN(close)) continue
    candles.push({
      time: time as Time,
      open,
      high,
      low,
      close,
      volume: isNaN(volume) ? 0 : volume,
    })
  }
  return candles
}

function splitRecord(line: string, delimiter: string): string[] {
  if (delimiter === '\t') return line.split('\t')
  const parts: string[] = []
  let current = ''
  let inQuotes = false
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]!
    if (ch === '"') {
      inQuotes = !inQuotes
    } else if (ch === ',' && !inQuotes) {
      parts.push(current)
      current = ''
    } else {
      current += ch
    }
  }
  parts.push(current)
  return parts
}

function parseOrdersTSV(raw: string): TradeMarker[] {
  const lines = raw.trim().split('\n')
  if (lines.length < 2) return []
  const delimiter = lines[0]!.includes('\t') ? '\t' : ','
  const header = splitRecord(lines[0]!, delimiter).map((h) => h.trim().toLowerCase())
  const sideIdx = header.indexOf('side')
  const priceIdx = header.indexOf('price')
  const qtyIdx = header.indexOf('quantity')
  const timeIdx = header.indexOf('creation_time') !== -1 ? header.indexOf('creation_time') : header.indexOf('time')
  if (sideIdx === -1 || priceIdx === -1 || qtyIdx === -1 || timeIdx === -1) return []

  const maxIdx = Math.max(sideIdx, priceIdx, qtyIdx, timeIdx)
  const markers: TradeMarker[] = []
  for (let i = 1; i < lines.length; i++) {
    const parts = splitRecord(lines[i]!, delimiter).map((p) => p.trim())
    if (parts.length <= maxIdx) continue
    const side = parts[sideIdx]!.toUpperCase()
    if (side !== 'BUY' && side !== 'SELL') continue
    const price = Number(parts[priceIdx])
    const quantity = Number(parts[qtyIdx])
    if (isNaN(price) || isNaN(quantity)) continue
    const rawTime = parts[timeIdx]!
    const ms = Number(rawTime) > 0 ? Number(rawTime) * 1000 : Date.parse(rawTime)
    const time = Math.floor(ms / 1000)
    if (isNaN(time) || time === 0) continue
    markers.push({ time: time as Time, side: side as 'BUY' | 'SELL', price, quantity })
  }
  return markers
}

async function fetchTSV(jobId: string, file: string): Promise<string> {
  const res = await fetch(`/api/manager/backtest/jobs/${jobId}/download?file=${file}`)
  if (!res.ok) throw new Error(`Failed to fetch ${file}: ${res.status}`)
  return res.text()
}

interface BacktestResultDisplayProps {
  output?: string
  report?: BacktestReport | null
  equityCurveTSV?: string
  jobId?: string
  symbol?: string
  exchange?: string
  startTime?: string
  endTime?: string
}

interface EquityTooltipProps {
  active?: boolean
  payload?: Array<{ value: number }>
  label?: string
  timeLabel: string
}

function EquityTooltip({ active, payload, label, timeLabel }: EquityTooltipProps) {
  if (!active || !payload?.length || label == null) return null
  const val = payload[0]!.value
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground">{timeLabel}{label}</p>
      <p className={`text-sm font-medium ${val >= 0 ? 'text-trade-up' : 'text-trade-down'}`}>
        {val >= 0 ? '+' : ''}{val.toFixed(2)}
      </p>
    </div>
  )
}

function n(v: unknown): number {
  const num = Number(v)
  if (!Number.isFinite(num)) return 0
  return num
}

function fmt(v: unknown, digits: number): string {
  return n(v).toFixed(digits)
}

function MetricCard({ label, value, isCurrency, highlight }: { label: string; value: string; isCurrency?: boolean; highlight?: boolean }) {
  const numVal = isCurrency ? parseFloat(value) : NaN
  const safeVal = Number.isFinite(numVal) ? numVal : 0
  const colorClass = isCurrency
    ? safeVal >= 0 ? 'text-trade-up' : 'text-trade-down'
    : highlight ? 'text-trade-up' : ''
  return (
    <div className="rounded-lg border bg-muted/30 px-3 py-2">
      <p className="text-[11px] text-muted-foreground">{label}</p>
      <p className={`text-sm font-semibold font-mono ${colorClass}`}>
        {isCurrency ? (safeVal >= 0 ? '+' : '') + safeVal.toFixed(2) : value}
      </p>
    </div>
  )
}

function snapToCandle(target: number, sortedTimes: number[]): number {
  if (sortedTimes.length === 0) return target
  let lo = 0
  let hi = sortedTimes.length - 1
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (sortedTimes[mid]! < target) lo = mid + 1
    else hi = mid
  }
  if (lo >= sortedTimes.length) return sortedTimes[sortedTimes.length - 1]!
  const prev = lo > 0 ? sortedTimes[lo - 1]! : null
  const next = sortedTimes[lo]!
  if (!prev) return next
  return (target - prev) <= (next - target) ? prev : next
}

const CHART_THEME: DeepPartial<ChartOptions> = {
  layout: {
    background: { type: ColorType.Solid, color: '#0c0e14' },
    textColor: '#798089',
  },
  grid: {
    vertLines: { color: '#1b1e26' },
    horzLines: { color: '#1b1e26' },
  },
  crosshair: {
    vertLine: { color: '#71717a', width: 1, style: 2 },
    horzLine: { color: '#71717a', width: 1, style: 2 },
  },
  rightPriceScale: { borderColor: '#1b1e26' },
  timeScale: { borderColor: '#1b1e26', timeVisible: true, secondsVisible: false },
}

function BacktestPriceChart({ jobId, exchange, symbol, startTime, endTime }: {
  jobId: string
  exchange?: string
  symbol?: string
  startTime?: string
  endTime?: string
}) {
  const t = useTranslations('Backtest')
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const markersPluginRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)
  const volumeSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const [loading, setLoading] = useState(true)
  const [hasData, setHasData] = useState(false)
  const dataKeyRef = useRef(jobId)

  const initChart = useCallback(() => {
    if (!containerRef.current) return
    if (chartRef.current) {
      try { chartRef.current.remove() } catch { /* disposed */ }
    }

    const chart = createChart(containerRef.current, {
      ...CHART_THEME,
      width: containerRef.current.clientWidth,
      height: 400,
    })

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    })

    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    })
    chart.priceScale('volume').applyOptions({ scaleMargins: { top: 0.85, bottom: 0 } })

    chartRef.current = chart
    candleSeriesRef.current = candleSeries
    volumeSeriesRef.current = volumeSeries
    markersPluginRef.current = null

    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        chart.applyOptions({ width: entry.contentRect.width })
      }
    })
    ro.observe(containerRef.current)

    return () => {
      ro.disconnect()
      if (markersPluginRef.current) {
        try { markersPluginRef.current.detach() } catch { /* ignore */ }
        markersPluginRef.current = null
      }
      chart.remove()
    }
  }, [])

  useEffect(() => {
    const cleanup = initChart()
    return () => cleanup?.()
  }, [initChart])

  useEffect(() => {
    if (dataKeyRef.current !== jobId) {
      dataKeyRef.current = jobId
    }
  }, [jobId])

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        const results = await Promise.all([
          exchange && symbol && startTime && endTime
            ? (() => {
                const startSec = Math.floor(new Date(startTime).getTime() / 1000)
                const endSec = Math.floor(new Date(endTime).getTime() / 1000)
                const days = Math.ceil((endSec - startSec) / 86400)
                const interval = days > 60 ? '4h' : '1h'
                const multiplier = days > 60 ? 4 : 1
                const count = Math.ceil((endSec - startSec) / 3600 / multiplier) + 10
                const limit = Math.min(count, 1500)
                return fetchMarketKlines(exchange, symbol, interval, limit,
                  startSec, endSec,
                ).then(r => r.klines).catch(() => [])
              })()
            : Promise.resolve([]),
          fetchTSV(jobId, 'orders').catch(() =>
            fetchTSV(jobId, 'trades').catch(() => '')
          ),
        ])
        if (cancelled) return

        const klines = results[0] as Array<{ time: number; open: string; high: string; low: string; close: string; volume: string }>
        const ordersTsv = results[1] as string

        if (klines.length === 0) {
          setLoading(false)
          return
        }

        const candles: KlineCandle[] = klines.map(k => ({
          time: (Math.floor(Number(k.time) / 1000)) as Time,
          open: Number(k.open),
          high: Number(k.high),
          low: Number(k.low),
          close: Number(k.close),
          volume: Number(k.volume),
        }))

        const series = candleSeriesRef.current
        const chart = chartRef.current
        if (!series || !chart) { setLoading(false); return }

        series.setData(candles.map(c => ({
          time: c.time, open: c.open, high: c.high, low: c.low, close: c.close,
        })))

        const volData = candles
          .filter(c => c.volume != null && c.volume > 0)
          .map(c => ({
            time: c.time,
            value: c.volume!,
            color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
          }))
        if (volData.length > 0 && volumeSeriesRef.current) {
          volumeSeriesRef.current.setData(volData)
        }

        if (candles.length > 80) {
          const ts = chart.timeScale()
          const start = candles[candles.length - 80]!
          const end = candles[candles.length - 1]!
          ts.setVisibleRange({ from: start.time, to: end.time })
        } else {
          chart.timeScale().fitContent()
        }

        if (ordersTsv) {
          const tradeMarkers = parseOrdersTSV(ordersTsv)
          if (tradeMarkers.length > 0) {
            const candleTimeSet = new Set(candles.map(c => c.time as number))
            const sortedTimes = [...candleTimeSet].sort((a, b) => a - b)
            const lwMarkers = tradeMarkers
              .sort((a, b) => (a.time as number) - (b.time as number))
              .map((m) => {
                const snapped = snapToCandle(m.time as number, sortedTimes)
                return {
                  time: snapped as Time,
                  position: (m.side === 'BUY' ? 'belowBar' : 'aboveBar') as 'belowBar' | 'aboveBar',
                  color: m.side === 'BUY' ? '#22c55e' : '#ef4444',
                  shape: (m.side === 'BUY' ? 'arrowUp' : 'arrowDown') as 'arrowUp' | 'arrowDown',
                  text: `${m.side === 'BUY' ? '▲' : '▼'} ${m.quantity}`,
                }
              })

            if (markersPluginRef.current) {
              markersPluginRef.current.setMarkers(lwMarkers)
            } else {
              markersPluginRef.current = createSeriesMarkers(series, lwMarkers)
            }
          }
        }

        setHasData(true)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    return () => { cancelled = true }
  }, [jobId, exchange, symbol, startTime, endTime])

  if (!loading && !hasData) return null

  return (
    <div className="relative">
      <div ref={containerRef} className="rounded-lg overflow-hidden" style={{ height: 400 }} />
      {loading && (
        <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-card/80">
          <span className="text-sm text-muted-foreground">{t('loading')}</span>
        </div>
      )}
    </div>
  )
}

export function BacktestResultDisplay({ output, report, equityCurveTSV, jobId, symbol, exchange, startTime, endTime }: BacktestResultDisplayProps) {
  const t = useTranslations('Backtest')

  const parsed = useMemo<ParsedBacktest>(() => {
    if (report) {
      const sr = report.symbolReports?.[0] ?? null
      let equityCurve: EquityPoint[] = []
      if (equityCurveTSV) {
        equityCurve = parseEquityCurveTSV(equityCurveTSV)
      }
      return { equityCurve, symbolReport: sr, report }
    }
    if (output) {
      return parseLegacyOutput(output)
    }
    return { equityCurve: [], symbolReport: null, report: null }
  }, [report, equityCurveTSV, output])

  if (!jobId && !parsed.symbolReport && parsed.equityCurve.length === 0) {
    return null
  }

  const sr = parsed.symbolReport
  const isTimeBased = parsed.equityCurve.length > 0 && !/^\d+$/.test(parsed.equityCurve[0]!.time)

  return (
    <div className="space-y-4">
      {jobId && <BacktestPriceChart jobId={jobId} exchange={exchange} symbol={symbol} startTime={startTime} endTime={endTime} />}

      {parsed.equityCurve.length > 0 && (
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={parsed.equityCurve}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
            <XAxis
              dataKey="time"
              tick={{ fontSize: 11 }}
              className="fill-muted-foreground"
              interval={Math.max(0, Math.floor(parsed.equityCurve.length / 7) - 1)}
              tickFormatter={(v: string) => {
                if (!isTimeBased) return v
                const d = new Date(v)
                if (isNaN(d.getTime())) return v
                return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
              }}
              padding={{ left: 10, right: 30 }}
              label={{ value: t(isTimeBased ? 'time' : 'tradeNumber'), position: 'insideBottomRight', offset: -5, fontSize: 10 }}
            />
            <YAxis
              tick={{ fontSize: 11 }}
              className="fill-muted-foreground"
              tickFormatter={(v: number) => v.toFixed(2)}
            />
            <Tooltip content={<EquityTooltip timeLabel={t(isTimeBased ? 'time' : 'tradeNumber')} />} />
            <ReferenceLine y={0} stroke="hsl(var(--border))" />
            <Area
              type="monotone"
              dataKey="value"
              stroke="hsl(221, 83%, 53%)"
              fill="hsl(221, 83%, 53%, 0.15)"
              strokeWidth={2}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      )}

      {sr && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          <MetricCard label={t('totalNetProfit')} value={fmt(sr.totalNetProfit, 2)} isCurrency />
          <MetricCard label={t('grossProfit')} value={fmt(sr.grossProfit, 2)} isCurrency />
          <MetricCard label={t('grossLoss')} value={fmt(sr.grossLoss, 2)} isCurrency />
          <MetricCard
            label={t('assetIncrease')}
            value={(() => {
              const init = n(parsed.report?.initialEquityValue) || 1
              const pct = (n(sr.totalNetProfit) / init) * 100
              const safe = Number.isFinite(pct) ? pct : 0
              return `${safe >= 0 ? '+' : ''}${safe.toFixed(2)}%`
            })()}
            highlight={n(sr.totalNetProfit) >= 0}
          />
          <MetricCard label={t('trades')} value={String(sr.tradeCount ?? 0)} />
          <MetricCard label={t('winRate')} value={`${(n(sr.percentProfitable) * 100).toFixed(1)}%`} highlight={n(sr.percentProfitable) >= 0.5} />
          <MetricCard label={t('sharpeRatio')} value={fmt(sr.sharpeRatio, 4)} />
          <MetricCard label={t('sortinoRatio')} value={fmt(sr.sortinoRatio, 4)} />
          <MetricCard label={t('profitFactor')} value={fmt(sr.profitFactor, 4)} />
          <MetricCard label={t('maxDrawdown')} value={`${(n(sr.maxDrawdown) * 100).toFixed(2)}%`} highlight={false} />
          <MetricCard label={t('winningTrades')} value={String(sr.winningCount ?? 0)} />
          <MetricCard label={t('losingTrades')} value={String(sr.losingCount ?? 0)} />
        </div>
      )}

      {!sr && parsed.report && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          <MetricCard label={t('totalProfit')} value={fmt(parsed.report.totalProfit, 2)} isCurrency />
          <MetricCard label={t('totalGrossProfit')} value={fmt(parsed.report.totalGrossProfit, 2)} isCurrency />
          <MetricCard label={t('totalGrossLoss')} value={fmt(parsed.report.totalGrossLoss, 2)} isCurrency />
        </div>
      )}
    </div>
  )
}
