'use client'

import { useState, useMemo, useCallback } from 'react'
import dynamic from 'next/dynamic'
import { cn } from '@/lib/utils'
import { ErrorBoundary } from '@/components/error-boundary'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { OhlcvLegend } from '@/components/chart/OhlcvLegend'
import type { TradeMarker, OrderLevel, GridLine, IndicatorLine, KlineCandle } from '@/components/chart/CandlestickChart'
import type { IndicatorConfig } from '@/lib/bbgo/indicators'
import type { StrategyStats } from '@/lib/bbgo/strategy-state'

const CandlestickChart = dynamic(
  () => import('@/components/chart/CandlestickChart').then((m) => ({ default: m.CandlestickChart })),
  { ssr: false, loading: () => <div className="h-[450px] animate-pulse rounded-lg bg-muted" /> }
)

const KLINE_INTERVALS = [
  { key: '1m', label: '1m' },
  { key: '5m', label: '5m' },
  { key: '15m', label: '15m' },
  { key: '1h', label: '1H' },
  { key: '4h', label: '4H' },
  { key: '1d', label: '1D' },
] as const

interface BotChartPanelProps {
  symbol: string
  exchange: string
  botReachable: boolean
  candles: KlineCandle[]
  tradeMarkers: TradeMarker[]
  orderLevels: OrderLevel[]
  gridLines: GridLine[]
  indicatorLines: IndicatorLine[]
  pnlLine: IndicatorLine | null
  klinesLoading: boolean
  loadEarlierKlines?: () => void
  strategyStats: StrategyStats | null
  currentPrice?: number
  noSymbolText: string
  startToSeeDataText: string
}

export function BotChartPanel({
  symbol,
  exchange,
  botReachable,
  candles,
  tradeMarkers,
  orderLevels,
  gridLines,
  indicatorLines,
  pnlLine,
  klinesLoading,
  loadEarlierKlines,
  strategyStats,
  currentPrice,
  noSymbolText,
  startToSeeDataText,
}: BotChartPanelProps) {
  const [klineInterval, setKlineInterval] = useState('1h')
  const [indicators, setIndicators] = useState<IndicatorConfig[]>([])
  const [showPnlCurve, setShowPnlCurve] = useState(true)
  const [ohlcvData, setOhlcvData] = useState<{
    time: number; open: number; high: number; low: number; close: number; volume?: number
  } | null>(null)

  const toggleIndicator = useCallback((id: string) => {
    setIndicators((prev) =>
      prev.map((ic) => (ic.id === id ? { ...ic, enabled: !ic.enabled } : ic))
    )
  }, [])

  const allIndicators = useMemo(() =>
    pnlLine && showPnlCurve ? [...indicatorLines, pnlLine] : indicatorLines,
    [indicatorLines, pnlLine, showPnlCurve]
  )

  return (
    <ErrorBoundary>
      <Card className="rounded-xl">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <CardTitle className="text-sm font-medium">
              {symbol || 'Price Chart'} · {exchange}
            </CardTitle>
            <div className="flex items-center gap-1 flex-wrap">
              {KLINE_INTERVALS.map((iv) => (
                <button
                  key={iv.key}
                  onClick={() => setKlineInterval(iv.key)}
                  className={cn(
                    'rounded px-2 py-0.5 text-xs font-medium transition-colors',
                    klineInterval === iv.key
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-muted-foreground hover:bg-muted/80'
                  )}
                >
                  {iv.label}
                </button>
              ))}
              <span className="mx-1 h-3 w-px bg-border" />
              {indicators.map((ic) => (
                <button
                  key={ic.id}
                  onClick={() => toggleIndicator(ic.id)}
                  className={cn(
                    'rounded px-1.5 py-0.5 text-[11px] font-medium transition-colors',
                    ic.enabled ? 'text-white' : 'text-muted-foreground hover:text-foreground'
                  )}
                  style={ic.enabled ? { backgroundColor: ic.color } : undefined}
                >
                  {ic.name}
                </button>
              ))}
              {pnlLine && (
                <button
                  onClick={() => setShowPnlCurve((v) => !v)}
                  className={cn(
                    'rounded px-1.5 py-0.5 text-[11px] font-medium transition-colors',
                    showPnlCurve ? 'text-white' : 'text-muted-foreground hover:text-foreground'
                  )}
                  style={showPnlCurve ? { backgroundColor: '#a855f7' } : undefined}
                >
                  P&L
                </button>
              )}
            </div>
          </div>
          {strategyStats && strategyStats.base > 0 && currentPrice && (
            <div className="flex items-center gap-4 mt-1.5 pt-2 border-t text-xs font-mono">
              <span className="text-muted-foreground">
                Pos: <span className="text-trade-up font-medium">{strategyStats.base.toFixed(6)}</span>
              </span>
              {strategyStats.averageCost > 0 && (
                <span className="text-muted-foreground">
                  Entry: <span className="text-foreground font-medium">{strategyStats.averageCost.toLocaleString()}</span>
                </span>
              )}
              <span className={cn(
                'font-medium',
                currentPrice > strategyStats.averageCost ? 'text-trade-up' : 'text-trade-down'
              )}>
                {currentPrice > strategyStats.averageCost ? '+' : ''}
                {((currentPrice - strategyStats.averageCost) * strategyStats.base).toFixed(2)} USDT
              </span>
              <span className="text-muted-foreground">
                ≈ {(currentPrice * strategyStats.base).toFixed(2)} USDT
              </span>
            </div>
          )}
        </CardHeader>
        <CardContent>
          {!symbol ? (
            <div className="flex h-[450px] items-center justify-center text-sm text-muted-foreground">
              {botReachable ? noSymbolText : startToSeeDataText}
            </div>
          ) : (
            <div className="flex gap-4">
              <div className="flex-1 min-w-0">
                <OhlcvLegend
                  data={ohlcvData}
                  symbol={symbol}
                  previousClose={candles.length > 1 ? candles[candles.length - 2]?.close : undefined}
                />
                <CandlestickChart
                  candles={candles}
                  tradeMarkers={tradeMarkers}
                  orderLevels={orderLevels}
                  gridLines={gridLines}
                  indicatorLines={allIndicators}
                  height={450}
                  isLoading={klinesLoading}
                  dataKey={`${exchange}-${symbol}-${klineInterval}`}
                  onVisibleTimeRangeChange={(range) => {
                    if (!range || candles.length === 0 || !candles[0]) return
                    const earliest = candles[0].time as number
                    const visibleSpan = range.to - range.from
                    if (range.from < earliest + visibleSpan * 0.5) {
                      loadEarlierKlines?.()
                    }
                  }}
                  onCandleHover={setOhlcvData}
                />
              </div>
              {strategyStats && (
                <StrategySidePanel strategyStats={strategyStats} currentPrice={currentPrice} gridLines={gridLines} />
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </ErrorBoundary>
  )
}

function StrategySidePanel({ strategyStats, currentPrice, gridLines }: {
  strategyStats: StrategyStats
  currentPrice?: number
  gridLines: GridLine[]
}) {
  return (
    <div className="hidden lg:flex flex-col gap-2 w-48 shrink-0 text-xs">
      <div className="rounded-lg border bg-muted/30 p-3 space-y-2">
        <p className="font-medium text-sm">{strategyStats.strategy}</p>
        <div className="space-y-1.5 font-mono text-muted-foreground">
          <div className="flex justify-between">
            <span>Range</span>
            <span className="text-foreground">{strategyStats.lowerPrice.toLocaleString()}–{strategyStats.upperPrice.toLocaleString()}</span>
          </div>
          <div className="flex justify-between">
            <span>Grids</span>
            <span className="text-foreground">{strategyStats.gridNumber}</span>
          </div>
          <div className="flex justify-between">
            <span>Qty/Grid</span>
            <span className="text-foreground">{strategyStats.quantity}</span>
          </div>
          {strategyStats.stopLossPrice > 0 && (
            <div className="flex justify-between">
              <span>Stop Loss</span>
              <span className="text-trade-down">{strategyStats.stopLossPrice.toLocaleString()}</span>
            </div>
          )}
          {strategyStats.takeProfitPrice > 0 && (
            <div className="flex justify-between">
              <span>Take Profit</span>
              <span className="text-trade-up">{strategyStats.takeProfitPrice.toLocaleString()}</span>
            </div>
          )}
          <hr className="border-border" />
        </div>
      </div>
      <div className="rounded-lg border bg-muted/30 p-3 space-y-2">
        <p className="font-medium text-sm">Position</p>
        <div className="space-y-1.5 font-mono text-muted-foreground">
          <div className="flex justify-between">
            <span>Base</span>
            <span className={cn('text-foreground', strategyStats.base > 0 && 'text-trade-up')}>
              {strategyStats.base.toFixed(6)}
            </span>
          </div>
          <div className="flex justify-between">
            <span>Quote</span>
            <span className={cn('text-foreground', strategyStats.quote > 0 && 'text-trade-up')}>
              {strategyStats.quote.toFixed(2)}
            </span>
          </div>
          {strategyStats.averageCost > 0 && (
            <div className="flex justify-between">
              <span>Avg Cost</span>
              <span className="text-foreground">{strategyStats.averageCost.toLocaleString()}</span>
            </div>
          )}
          {currentPrice && strategyStats.base > 0 && strategyStats.averageCost > 0 && (
            <div className="flex justify-between">
              <span>Unrealized</span>
              <span className={cn(
                'font-medium',
                currentPrice > strategyStats.averageCost ? 'text-trade-up' : 'text-trade-down'
              )}>
                {currentPrice > strategyStats.averageCost ? '+' : ''}
                {((currentPrice - strategyStats.averageCost) * strategyStats.base).toFixed(2)} USDT
              </span>
            </div>
          )}
          {currentPrice && strategyStats.base > 0 && (
            <div className="flex justify-between">
              <span>Value</span>
              <span className="text-foreground">{(currentPrice * strategyStats.base).toFixed(2)}</span>
            </div>
          )}
        </div>
      </div>
      {gridLines.length > 0 && (
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="font-medium mb-1.5">Grid Levels</p>
          <div className="max-h-40 overflow-y-auto space-y-0.5 font-mono text-muted-foreground">
            {gridLines.slice(0, 12).map((g, i) => (
              <div key={i} className="flex justify-between">
                <span className="text-[10px]">{g.label}</span>
                <span className="w-1.5 h-1.5 rounded-full mt-1.5" style={{ backgroundColor: g.color }} />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
