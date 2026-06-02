'use client'

import { useMemo } from 'react'
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

interface BacktestEquityChartProps {
  output: string
}

interface EquityPoint {
  trade: number
  cumulativePnl: number
}

interface BacktestMetrics {
  totalReturn: number
  unrealizedReturn: number
  assetIncrease: number
  assetIncreasePercent: number
  tradeCount: number
  sharpeRatio: number
  sortinoRatio: number
}

interface ParsedBacktest {
  equityCurve: EquityPoint[]
  metrics: BacktestMetrics | null
}

const REALIZED_PROFIT_PATTERN = /REALIZED\s+PROFIT:\s*([-+]?[\d.,]+)\s+(\S+)/i
const UNREALIZED_PROFIT_PATTERN = /UNREALIZED\s+PROFIT:\s*([-+]?[\d.,]+)\s+(\S+)/i
const ASSET_INCREASED_PATTERN = /ASSET\s+(?:INCREASED|DECREASED):\s*([-+]?[\d.,]+)\s+\S+\s+\(([+-]?[\d.]+)%\)/i
const TRADE_COUNT_PATTERN = /NUMBER\s+OF\s+TRADES:\s*(\d+)/i
const SHARPE_PATTERN = /REALIZED\s+SHARPE\s+RATIO:\s*([-+]?[\d.,]+)/i
const SORTINO_PATTERN = /REALIZED\s+SORTINO\s+RATIO:\s*([-+]?[\d.,]+)/i
const TRADE_PROFIT_PATTERN = /trade\s+#?(\d+).*?profit[:\s]+([-+]?[\d.,]+)/gi

function parseBacktestOutput(output: string): ParsedBacktest {
  const cleanOutput = output.replace(/\x1b\[[0-9;]*m/g, '')

  const metrics: BacktestMetrics = {
    totalReturn: 0,
    unrealizedReturn: 0,
    assetIncrease: 0,
    assetIncreasePercent: 0,
    tradeCount: 0,
    sharpeRatio: 0,
    sortinoRatio: 0,
  }

  const rpMatch = cleanOutput.match(REALIZED_PROFIT_PATTERN)
  if (rpMatch) metrics.totalReturn = parseFloat(rpMatch[1]!.replace(/,/g, ''))

  const upMatch = cleanOutput.match(UNREALIZED_PROFIT_PATTERN)
  if (upMatch) metrics.unrealizedReturn = parseFloat(upMatch[1]!.replace(/,/g, ''))

  const aiMatch = cleanOutput.match(ASSET_INCREASED_PATTERN)
  if (aiMatch) {
    metrics.assetIncrease = parseFloat(aiMatch[1]!.replace(/,/g, ''))
    metrics.assetIncreasePercent = parseFloat(aiMatch[2]!)
  }

  const tcMatch = cleanOutput.match(TRADE_COUNT_PATTERN)
  if (tcMatch) metrics.tradeCount = parseInt(tcMatch[1]!, 10)

  const shMatch = cleanOutput.match(SHARPE_PATTERN)
  if (shMatch) metrics.sharpeRatio = parseFloat(shMatch[1]!.replace(/,/g, ''))

  const soMatch = cleanOutput.match(SORTINO_PATTERN)
  if (soMatch) metrics.sortinoRatio = parseFloat(soMatch[1]!.replace(/,/g, ''))

  const equityCurve: EquityPoint[] = []
  let cumulative = 0
  let match: RegExpExecArray | null
  const regex = new RegExp(TRADE_PROFIT_PATTERN.source, 'gi')
  let tradeNum = 0

  while ((match = regex.exec(cleanOutput)) !== null) {
    tradeNum++
    const profit = parseFloat(match[2]!.replace(/,/g, ''))
    cumulative += profit
    equityCurve.push({ trade: tradeNum, cumulativePnl: Math.round(cumulative * 100) / 100 })
  }

  const hasData = metrics.tradeCount > 0 || metrics.totalReturn !== 0 || equityCurve.length > 0
  return { equityCurve, metrics: hasData ? metrics : null }
}

interface EquityTooltipProps {
  active?: boolean
  payload?: Array<{ value: number }>
  label?: number
  tradeLabel: string
}

function EquityTooltip({ active, payload, label, tradeLabel }: EquityTooltipProps) {
  if (!active || !payload?.length || label == null) return null
  const val = payload[0]!.value
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground">{tradeLabel}{label}</p>
      <p className={`text-sm font-medium ${val >= 0 ? 'text-trade-up' : 'text-trade-down'}`}>
        {val >= 0 ? '+' : ''}{val.toFixed(2)}
      </p>
    </div>
  )
}

export function BacktestEquityChart({ output }: BacktestEquityChartProps) {
  const t = useTranslations('Backtest')
  const parsed = useMemo(() => parseBacktestOutput(output), [output])

  if (!parsed.metrics && parsed.equityCurve.length === 0) {
    return null
  }

  return (
    <div className="space-y-4">
      {parsed.equityCurve.length > 0 && (
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={parsed.equityCurve}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
            <XAxis
              dataKey="trade"
              tick={{ fontSize: 11 }}
              className="fill-muted-foreground"
              label={{ value: t('tradeNumber'), position: 'insideBottomRight', offset: -5, fontSize: 10 }}
            />
            <YAxis
              tick={{ fontSize: 11 }}
              className="fill-muted-foreground"
              tickFormatter={(v: number) => v.toFixed(2)}
            />
            <Tooltip content={<EquityTooltip tradeLabel={t('tradeNumber')} />} />
            <ReferenceLine y={0} stroke="hsl(var(--border))" />
            <Area
              type="monotone"
              dataKey="cumulativePnl"
              stroke="hsl(221, 83%, 53%)"
              fill="hsl(221, 83%, 53%, 0.15)"
              strokeWidth={2}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      )}

      {parsed.metrics && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          <MetricCard label={t('realizedPnl')} value={String(parsed.metrics.totalReturn)} isCurrency />
          <MetricCard label={t('unrealizedPnl')} value={String(parsed.metrics.unrealizedReturn)} isCurrency />
          <MetricCard label={t('assetIncrease')} value={`${parsed.metrics.assetIncreasePercent >= 0 ? '+' : ''}${parsed.metrics.assetIncreasePercent.toFixed(2)}%`} highlight={parsed.metrics.assetIncreasePercent >= 0} />
          <MetricCard label={t('trades')} value={String(parsed.metrics.tradeCount)} />
          <MetricCard label={t('sharpeRatio')} value={parsed.metrics.sharpeRatio.toFixed(4)} />
          <MetricCard label={t('sortinoRatio')} value={parsed.metrics.sortinoRatio.toFixed(4)} />
        </div>
      )}
    </div>
  )
}

function MetricCard({ label, value, isCurrency, highlight }: { label: string; value: string; isCurrency?: boolean; highlight?: boolean }) {
  const numVal = isCurrency ? parseFloat(value) : NaN
  const colorClass = isCurrency
    ? numVal >= 0 ? 'text-trade-up' : 'text-trade-down'
    : highlight ? 'text-trade-up' : ''
  return (
    <div className="rounded-lg border bg-muted/30 px-3 py-2">
      <p className="text-[11px] text-muted-foreground">{label}</p>
      <p className={`text-sm font-semibold font-mono ${colorClass}`}>
        {isCurrency ? (numVal >= 0 ? '+' : '') + numVal.toFixed(2) : value}
      </p>
    </div>
  )
}
