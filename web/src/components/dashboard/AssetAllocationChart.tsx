'use client'

import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts'
import type { BBGoAsset } from '@/lib/bbgo/queries'

interface AssetAllocationChartProps {
  assets: Record<string, BBGoAsset>
}

const COLORS = [
  'hsl(221, 83%, 53%)',
  'hsl(142, 71%, 45%)',
  'hsl(38, 92%, 50%)',
  'hsl(0, 84%, 60%)',
  'hsl(262, 83%, 58%)',
  'hsl(186, 78%, 45%)',
  'hsl(326, 100%, 74%)',
  'hsl(24, 95%, 53%)',
]

interface ChartEntry {
  name: string
  value: number
  usdValue: string
}

function buildChartData(assets: Record<string, BBGoAsset>): ChartEntry[] {
  const entries = Object.values(assets ?? {})
    .map((a) => ({
      name: a.currency,
      value: parseFloat(a.netAssetInUSD || '0'),
      usdValue: a.netAssetInUSD,
    }))
    .filter((e) => e.value > 0)
    .sort((a, b) => b.value - a.value)

  const top = entries.slice(0, 7)
  const otherValue = entries.slice(7).reduce((s, e) => s + e.value, 0)
  if (otherValue > 0) {
    top.push({ name: 'Other', value: otherValue, usdValue: otherValue.toFixed(2) })
  }
  return top
}

function CustomTooltip({ active, payload }: { active?: boolean; payload?: Array<{ payload: ChartEntry }> }) {
  if (!active || !payload?.length) return null
  const entry = payload[0]?.payload
  if (!entry) return null
  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-sm font-medium">{entry.name}</p>
      <p className="text-sm text-muted-foreground">${entry.usdValue}</p>
    </div>
  )
}

export function AssetAllocationChart({ assets }: AssetAllocationChartProps) {
  const data = buildChartData(assets)

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-muted-foreground text-sm">
        No asset data available
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <PieChart>
        <Pie
          data={data}
          dataKey="value"
          nameKey="name"
          cx="50%"
          cy="50%"
          innerRadius={60}
          outerRadius={100}
          paddingAngle={2}
          strokeWidth={0}
        >
          {data.map((_, i) => (
            <Cell key={i} fill={COLORS[i % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip content={<CustomTooltip />} />
        <Legend
          verticalAlign="bottom"
          height={36}
          formatter={(value: string) => (
            <span className="text-xs text-foreground">{value}</span>
          )}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}
