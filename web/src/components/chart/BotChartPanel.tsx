'use client'

import { useState, useMemo } from 'react'
import dynamic from 'next/dynamic'
import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { ErrorBoundary } from '@/components/error-boundary'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { OhlcvLegend } from '@/components/chart/OhlcvLegend'
import type { TradeMarker, OrderLevel, GridLine, IndicatorLine, KlineCandle } from '@/components/chart/CandlestickChart'
import type { IndicatorConfig } from '@/lib/bbgo/indicators'
import type { StrategyDetails } from '@/lib/bbgo/strategy-state'
import { extractBaseCurrency } from '@/lib/bbgo/fifo-pnl'
import { StrategySidePanel } from './strategy-panel'

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
  strategyStats: StrategyDetails | null
  currentPrice?: number
  unrealizedPnlFromReport?: number
  noSymbolText: string
  startToSeeDataText: string
  klineInterval: string
  onIntervalChange: (interval: string) => void
  indicatorConfigs?: IndicatorConfig[]
  onToggleIndicator?: (id: string) => void
  supabasePosition?: {
    base: number
    averageCost: number
    quote: number
    symbol: string
    isClosed: boolean
    leverage?: number
    liquidationPrice?: number
    direction?: 'long' | 'short' | 'flat'
  }
  unrealizedPnlPct?: number
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
  unrealizedPnlFromReport,
  noSymbolText,
  startToSeeDataText,
  klineInterval,
  onIntervalChange,
  indicatorConfigs = [],
  onToggleIndicator,
  supabasePosition,
  unrealizedPnlPct,
}: BotChartPanelProps) {
  const t = useTranslations('Bots')
  const sp = useTranslations('Bots.chartSidePanel')
  const [showPnlCurve, setShowPnlCurve] = useState(true)
  const [ohlcvData, setOhlcvData] = useState<{
    time: number; open: number; high: number; low: number; close: number; volume?: number
  } | null>(null)

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
              {symbol || t('priceChart')} · {exchange}
            </CardTitle>
            <div className="flex items-center gap-1 flex-wrap">
              {KLINE_INTERVALS.map((iv) => (
                <button
                  key={iv.key}
                  onClick={() => onIntervalChange(iv.key)}
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
              {indicatorConfigs.map((ic) => (
                <button
                  key={ic.id}
                  onClick={() => onToggleIndicator?.(ic.id)}
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
          {supabasePosition && !supabasePosition.isClosed && currentPrice && (
            <div className="flex items-center gap-4 mt-1.5 pt-2 border-t text-xs font-mono">
              {supabasePosition.direction && supabasePosition.direction !== 'flat' && (
                <span className={cn(
                  'rounded px-1.5 py-0.5 text-[10px] font-bold uppercase',
                  supabasePosition.direction === 'long' ? 'bg-trade-up/20 text-trade-up' : 'bg-trade-down/20 text-trade-down'
                )}>
                  {supabasePosition.direction === 'long' ? '▲ Long' : '▼ Short'}
                </span>
              )}
              {supabasePosition.leverage && supabasePosition.leverage > 1 && (
                <span className="text-amber-400 font-semibold">
                  {supabasePosition.leverage}x
                </span>
              )}
              <span className="text-muted-foreground">
                {sp('holding')}: <span className="text-trade-up font-medium">{Math.abs(supabasePosition.base).toFixed(6)} {extractBaseCurrency(supabasePosition.symbol)}</span>
              </span>
              {supabasePosition.averageCost > 0 && (
                <span className="text-muted-foreground">
                  {sp('entry')}: <span className="text-foreground font-medium">${supabasePosition.averageCost.toLocaleString()}</span>
                </span>
              )}
              {supabasePosition.liquidationPrice && supabasePosition.liquidationPrice > 0 && (
                <span className="text-trade-down/80">
                  Liq: <span className="font-medium">${supabasePosition.liquidationPrice.toLocaleString()}</span>
                  <span className="text-muted-foreground ml-1">
                    ({((Math.abs(currentPrice - supabasePosition.liquidationPrice) / currentPrice) * 100).toFixed(1)}%)
                  </span>
                </span>
              )}
              <span className={cn(
                'font-medium',
                (unrealizedPnlFromReport ?? 0) >= 0 ? 'text-trade-up' : 'text-trade-down'
              )}>
                {(unrealizedPnlFromReport ?? 0) >= 0 ? '+' : ''}{(unrealizedPnlFromReport ?? 0).toFixed(2)} USDT
              </span>
              <span className="text-muted-foreground">
                ≈ ${(currentPrice * Math.abs(supabasePosition.base)).toFixed(2)}
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
                  height={520}
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
                  liquidationPrice={supabasePosition?.liquidationPrice}
                />
              </div>
              {strategyStats && (
                <StrategySidePanel details={strategyStats} currentPrice={currentPrice} gridLines={gridLines} unrealizedPnlFromReport={unrealizedPnlFromReport} supabasePosition={supabasePosition} unrealizedPnlPct={unrealizedPnlPct} />
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </ErrorBoundary>
  )
}
