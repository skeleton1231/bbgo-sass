'use client'

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
import type { PnlCurvePoint } from '@/lib/bbgo/queries'

interface EquityChartProps {
  pnlCurve: PnlCurvePoint[]
}

interface EquityTooltipProps {
  active?: boolean
  payload?: Array<{ value: number }>
  label?: string
}

function EquityTooltip({ active, payload, label }: EquityTooltipProps) {
  if (!active || !payload?.length || !label) return null
  const val = payload[0]!.value
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className={`text-sm font-medium font-mono ${val >= 0 ? 'text-trade-up' : 'text-trade-down'}`}>
        {val >= 0 ? '+' : ''}{val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
      </p>
    </div>
  )
}

export function EquityChart({ pnlCurve }: EquityChartProps) {
  const t = useTranslations('Dashboard')

  if (pnlCurve.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-muted-foreground text-sm">
        {t('noEquityData')}
      </div>
    )
  }

  const data = pnlCurve.map((p) => ({
    date: new Date(p.time * 1000).toISOString().slice(0, 10),
    cumulativePnl: p.value,
  }))

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
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
          tickFormatter={(v: number) => `$${v.toFixed(2)}`}
        />
        <Tooltip content={<EquityTooltip />} />
        <ReferenceLine y={0} stroke="hsl(var(--border))" />
        <Area
          type="stepAfter"
          dataKey="cumulativePnl"
          stroke="hsl(221, 83%, 53%)"
          fill="hsl(221, 83%, 53%, 0.15)"
          strokeWidth={2}
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
