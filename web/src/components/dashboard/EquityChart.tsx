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
} from 'recharts'
import type { BBGoAsset } from '@/lib/bbgo/queries'

interface EquityChartProps {
  assets: Record<string, BBGoAsset>
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
      <p className="text-sm font-medium font-mono">
        ${val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
      </p>
    </div>
  )
}

export function EquityChart({ assets }: EquityChartProps) {
  const t = useTranslations('Dashboard')

  const data = useMemo(() => {
    return Object.values(assets)
      .filter((a) => parseFloat(a.netAssetInUSD || '0') > 0)
      .map((a) => ({
        currency: a.currency,
        total: parseFloat(a.total || '0'),
        netAssetInUSD: parseFloat(a.netAssetInUSD || '0'),
      }))
      .sort((a, b) => b.netAssetInUSD - a.netAssetInUSD)
  }, [assets])

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-muted-foreground text-sm">
        {t('noEquityData')}
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
        <XAxis
          dataKey="currency"
          tick={{ fontSize: 11 }}
          className="fill-muted-foreground"
        />
        <YAxis
          tick={{ fontSize: 11 }}
          className="fill-muted-foreground"
          tickFormatter={(v: number) => `$${(v / 1000).toFixed(1)}k`}
        />
        <Tooltip content={<EquityTooltip />} />
        <Area
          type="monotone"
          dataKey="netAssetInUSD"
          stroke="hsl(221, 83%, 53%)"
          fill="hsl(221, 83%, 53%, 0.15)"
          strokeWidth={2}
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
