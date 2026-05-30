'use client'

import { cn } from '@/lib/utils'

interface OhlcvData {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume?: number
}

interface OhlcvLegendProps {
  data: OhlcvData | null
  symbol?: string
  previousClose?: number
}

function fmt(v: number, digits = 2): string {
  return v.toLocaleString('en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

function fmtVol(v: number): string {
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(2)}M`
  if (v >= 1_000) return `${(v / 1_000).toFixed(2)}K`
  return v.toFixed(2)
}

export function OhlcvLegend({ data, symbol, previousClose }: OhlcvLegendProps) {
  if (!data) {
    return (
      <div className="flex items-center gap-4 px-1 pb-2 font-mono text-xs text-muted-foreground">
        {symbol && <span className="font-semibold text-foreground">{symbol}</span>}
        <span>Hover chart to see OHLCV</span>
      </div>
    )
  }

  const change = data.close - data.open
  const isUp = change >= 0

  const refClose = previousClose ?? data.open
  const refChange = data.close - refClose
  const refChangePct = refClose > 0 ? (refChange / refClose) * 100 : 0
  const refIsUp = refChange >= 0

  return (
    <div className="flex items-center gap-3 px-1 pb-2 font-mono text-xs overflow-x-auto">
      {symbol && <span className="font-semibold text-foreground shrink-0">{symbol}</span>}

      <span className="text-muted-foreground shrink-0">O</span>
      <span className="text-foreground">{fmt(data.open)}</span>

      <span className="text-muted-foreground shrink-0">H</span>
      <span className={cn('text-foreground', data.high > data.open && 'text-trade-up', data.high < data.open && 'text-trade-down')}>
        {fmt(data.high)}
      </span>

      <span className="text-muted-foreground shrink-0">L</span>
      <span className={cn('text-foreground', data.low > data.open && 'text-trade-up', data.low < data.open && 'text-trade-down')}>
        {fmt(data.low)}
      </span>

      <span className="text-muted-foreground shrink-0">C</span>
      <span className={cn('text-foreground font-medium', isUp ? 'text-trade-up' : 'text-trade-down')}>
        {fmt(data.close)}
      </span>

      {data.volume != null && (
        <>
          <span className="text-muted-foreground shrink-0">Vol</span>
          <span className="text-foreground">{fmtVol(data.volume)}</span>
        </>
      )}

      <div className={cn(
        'flex items-center gap-1 rounded px-1.5 py-0.5 shrink-0',
        refIsUp ? 'bg-trade-up/10 text-trade-up' : 'bg-trade-down/10 text-trade-down'
      )}>
        <span>{refIsUp ? '+' : ''}{refChange.toFixed(2)}</span>
        <span>({refIsUp ? '+' : ''}{refChangePct.toFixed(2)}%)</span>
      </div>
    </div>
  )
}
