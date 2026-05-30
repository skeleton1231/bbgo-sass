'use client'

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from 'recharts'
import { useTranslations } from 'next-intl'

interface DepthChartProps {
  bids: Array<{ price: number; volume: number }>
  asks: Array<{ price: number; volume: number }>
  height?: number
}

function buildDepthData(bids: Array<{ price: number; volume: number }>, asks: Array<{ price: number; volume: number }>) {
  const sortedBids = [...bids].sort((a, b) => a.price - b.price)
  const sortedAsks = [...asks].sort((a, b) => a.price - b.price)

  let bidCumulative = 0
  const bidData = sortedBids.map((b) => {
    bidCumulative += b.volume
    return { price: b.price, bids: bidCumulative, asks: 0 }
  })

  let askCumulative = 0
  const askData = sortedAsks.map((a) => {
    askCumulative += a.volume
    return { price: a.price, bids: 0, asks: askCumulative }
  })

  const all = [...bidData, ...askData].sort((a, b) => a.price - b.price)

  let runningBids = 0
  let runningAsks = 0
  return all.map((d) => {
    if (d.bids > 0) runningBids = d.bids
    if (d.asks > 0) runningAsks = d.asks
    return { price: d.price, bids: runningBids, asks: runningAsks }
  })
}

interface DepthTooltipProps {
  active?: boolean
  payload?: Array<{ value: number; dataKey: string }>
  label?: number
}

function DepthTooltip({ active, payload, label }: DepthTooltipProps) {
  const t = useTranslations('Bots')
  if (!active || !payload?.length || label == null) return null
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground mb-1">{label.toFixed(2)}</p>
      {payload.map((p) => (
        <p key={p.dataKey} className="text-xs" style={{ color: p.dataKey === 'bids' ? '#22c55e' : '#ef4444' }}>
          {p.dataKey === 'bids' ? t('bids') : t('asks')}: {p.value.toFixed(4)}
        </p>
      ))}
    </div>
  )
}

export function DepthChart({ bids, asks, height = 300 }: DepthChartProps) {
  const t = useTranslations('Bots')
  if (bids.length === 0 && asks.length === 0) {
    return (
      <div className="flex items-center justify-center rounded-lg border bg-card" style={{ height }}>
        <span className="text-sm text-muted-foreground">{t('noDepthData')}</span>
      </div>
    )
  }

  const data = buildDepthData(bids, asks)

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
        <XAxis
          dataKey="price"
          tick={{ fontSize: 10 }}
          className="fill-muted-foreground"
          tickFormatter={(v: number) => v.toFixed(2)}
        />
        <YAxis
          tick={{ fontSize: 10 }}
          className="fill-muted-foreground"
          tickFormatter={(v: number) => v.toFixed(2)}
        />
        <Tooltip content={<DepthTooltip />} />
        <Area
          type="stepAfter"
          dataKey="bids"
          stroke="#22c55e"
          fill="rgba(34, 197, 94, 0.15)"
          strokeWidth={1.5}
          isAnimationActive={false}
        />
        <Area
          type="stepAfter"
          dataKey="asks"
          stroke="#ef4444"
          fill="rgba(239, 68, 68, 0.15)"
          strokeWidth={1.5}
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
