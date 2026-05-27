'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from 'recharts'
import { cn } from '@/lib/utils'

interface TradingVolumeEntry {
  year: number
  month?: number
  day?: number
  time?: string
  exchange?: string
  symbol?: string
  quoteVolume: number
}

interface TradingVolumeChartProps {
  volumes: TradingVolumeEntry[]
}

const PERIOD_KEYS = ['period7d', 'period30d', 'periodMonth', 'periodYear'] as const
const PERIOD_VALUES = ['7d', '30d', 'month', 'year'] as const

function formatLabel(entry: TradingVolumeEntry, period: string): string {
  if (period === 'year') return String(entry.year)
  if (period === 'month') return `${entry.year}-${String(entry.month).padStart(2, '0')}`
  if (entry.day && entry.month) return `${entry.month}/${entry.day}`
  return String(entry.year)
}

function CustomTooltip({ active, payload, label }: { active?: boolean; payload?: Array<{ value: number }>; label?: string }) {
  if (!active || !payload?.length) return null
  const val = payload[0]?.value
  if (val == null) return null
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-medium">${val.toLocaleString(undefined, { maximumFractionDigits: 2 })}</p>
    </div>
  )
}

export function TradingVolumeChart({ volumes }: TradingVolumeChartProps) {
  const t = useTranslations('Dashboard')
  const [period, setPeriod] = useState('7d')

  if (!volumes || volumes.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-muted-foreground text-sm">
        {t('noVolumeData')}
      </div>
    )
  }

  const chartData = volumes.map((v) => ({
    label: formatLabel(v, period),
    volume: v.quoteVolume,
  }))

  return (
    <div>
      <div className="flex gap-1 mb-4">
        {PERIOD_KEYS.map((key, i) => {
          const value = PERIOD_VALUES[i]!
          return (
            <button
              key={value}
              onClick={() => setPeriod(value)}
              className={cn(
                'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
                period === value
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-muted/80'
              )}
            >
              {t(key)}
            </button>
          )
        })}
      </div>
      <ResponsiveContainer width="100%" height={260}>
        <BarChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
          <XAxis dataKey="label" tick={{ fontSize: 11 }} className="fill-muted-foreground" />
          <YAxis
            tick={{ fontSize: 11 }}
            className="fill-muted-foreground"
            tickFormatter={(v: number) => `$${(v / 1000).toFixed(0)}k`}
          />
          <Tooltip content={<CustomTooltip />} />
          <Bar dataKey="volume" fill="hsl(221, 83%, 53%)" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
