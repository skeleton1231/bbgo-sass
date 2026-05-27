'use client'

import { useMemo } from 'react'
import { useTranslations } from 'next-intl'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  ReferenceLine,
  Cell,
} from 'recharts'
import type { BBGoTrade } from '@/lib/bbgo/queries'

interface PnlChartProps {
  trades: BBGoTrade[]
}

interface DailyPnl {
  date: string
  pnl: number
}

function computeDailyPnl(trades: BBGoTrade[]): DailyPnl[] {
  const byDate = new Map<string, number>()
  for (const trade of trades) {
    if (!trade.tradedAt) continue
    const day = trade.tradedAt.slice(0, 10)
    const qty = parseFloat(trade.quantity)
    const price = parseFloat(trade.price)
    const fee = parseFloat(trade.fee || '0')
    const sign = trade.side === 'SELL' ? 1 : -1
    const value = sign * qty * price - fee
    byDate.set(day, (byDate.get(day) ?? 0) + value)
  }
  return Array.from(byDate.entries())
    .map(([date, pnl]) => ({ date, pnl: Math.round(pnl * 100) / 100 }))
    .sort((a, b) => a.date.localeCompare(b.date))
}

interface PnlTooltipProps {
  active?: boolean
  payload?: Array<{ value: number }>
  label?: string
}

function PnlTooltip({ active, payload, label }: PnlTooltipProps) {
  if (!active || !payload?.length || !label) return null
  const val = payload[0]!.value
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className={`text-sm font-medium ${val >= 0 ? 'text-trade-up' : 'text-trade-down'}`}>
        {val >= 0 ? '+' : ''}{val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
      </p>
    </div>
  )
}

export function PnlChart({ trades }: PnlChartProps) {
  const t = useTranslations('Dashboard')

  const data = useMemo(() => computeDailyPnl(trades), [trades])

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-muted-foreground text-sm">
        {t('noPnlData')}
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
        <XAxis
          dataKey="date"
          tick={{ fontSize: 11 }}
          className="fill-muted-foreground"
          tickFormatter={(v: string) => v.slice(5)}
        />
        <YAxis
          tick={{ fontSize: 11 }}
          className="fill-muted-foreground"
          tickFormatter={(v: number) => `$${v.toFixed(0)}`}
        />
        <Tooltip content={<PnlTooltip />} />
        <ReferenceLine y={0} stroke="hsl(var(--border))" />
        <Bar dataKey="pnl" radius={[4, 4, 0, 0]} isAnimationActive={false}>
          {data.map((entry, i) => (
            <Cell key={i} fill={entry.pnl >= 0 ? '#22c55e' : '#ef4444'} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )
}
