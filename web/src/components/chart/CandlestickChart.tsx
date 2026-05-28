'use client'

import { useEffect, useRef, useCallback, useState } from 'react'
import {
  createChart,
  createSeriesMarkers,
  CandlestickSeries,
  HistogramSeries,
  LineSeries,
  type IChartApi,
  type ISeriesApi,
  type SeriesType,
  type Time,
  type DeepPartial,
  type ChartOptions,
  type ISeriesMarkersPluginApi,
  type LineStyle,
  ColorType,
} from 'lightweight-charts'

export interface KlineCandle {
  time: Time
  open: number
  high: number
  low: number
  close: number
  volume?: number
}

export interface TradeMarker {
  time: Time
  side: 'BUY' | 'SELL'
  price: number
  quantity: number
  isMaker?: boolean
  fee?: string
  feeCurrency?: string
  orderId?: number
  positionAction?: 'open' | 'close' | 'add' | 'reduce' | 'trade'
}

export interface OrderLevel {
  price: number
  side: 'BUY' | 'SELL'
  quantity: string
}

export interface GridLine {
  price: number
  label: string
  color: string
}

export interface IndicatorLine {
  id: string
  name: string
  color: string
  lineWidth?: number
  lineStyle?: number
  priceScaleId?: string
  scaleMargins?: { top: number; bottom: number }
  data: Array<{ time: Time; value: number }>
}

interface CandlestickChartProps {
  candles: KlineCandle[]
  tradeMarkers?: TradeMarker[]
  orderLevels?: OrderLevel[]
  gridLines?: GridLine[]
  indicatorLines?: IndicatorLine[]
  height?: number
  isLoading?: boolean
  dataKey?: string
  onVisibleTimeRangeChange?: (range: { from: number; to: number } | null) => void
  onCrosshairMove?: (data: { time?: number; price?: number } | null) => void
  onCandleHover?: (data: { time: number; open: number; high: number; low: number; close: number; volume?: number } | null) => void
}

function getChartTheme(): DeepPartial<ChartOptions> {
  return {
    layout: {
      background: { type: ColorType.Solid, color: '#0c0e14' },
      textColor: '#798089',
    },
    grid: {
      vertLines: { color: '#1b1e26' },
      horzLines: { color: '#1b1e26' },
    },
    crosshair: {
      vertLine: { color: '#71717a', width: 1, style: 2 },
      horzLine: { color: '#71717a', width: 1, style: 2 },
    },
    rightPriceScale: {
      borderColor: '#1b1e26',
    },
    timeScale: {
      borderColor: '#1b1e26',
      timeVisible: true,
      secondsVisible: false,
    },
  }
}

export function CandlestickChart({
  candles,
  tradeMarkers,
  orderLevels,
  gridLines,
  indicatorLines,
  height = 400,
  isLoading,
  dataKey,
  onVisibleTimeRangeChange,
  onCrosshairMove,
  onCandleHover,
}: CandlestickChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const volumeSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const indicatorSeriesRef = useRef<Map<string, ISeriesApi<SeriesType>>>(new Map())
  const markersRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)
  const priceLinesRef = useRef<ReturnType<ISeriesApi<SeriesType>['createPriceLine']>[]>([])
  const prevCandleCountRef = useRef(0)
  const prevDataKeyRef = useRef(dataKey)
  const onVisibleRangeChangeRef = useRef(onVisibleTimeRangeChange)
  const onCrosshairMoveRef = useRef(onCrosshairMove)
  const [tooltip, setTooltip] = useState<{ x: number; y: number; data: TradeMarker } | null>(null)
  onVisibleRangeChangeRef.current = onVisibleTimeRangeChange
  onCrosshairMoveRef.current = onCrosshairMove
  const onCandleHoverRef = useRef(onCandleHover)
  onCandleHoverRef.current = onCandleHover
  const prevMarkersKeyRef = useRef('')

  if (prevDataKeyRef.current !== dataKey) {
    prevDataKeyRef.current = dataKey
    prevCandleCountRef.current = 0
  }

  const initChart = useCallback(() => {
    if (!containerRef.current) return

    if (chartRef.current) {
      try { chartRef.current.remove() } catch { /* already disposed */ }
      chartRef.current = null
    }

    const chart = createChart(containerRef.current, {
      ...getChartTheme(),
      width: containerRef.current.clientWidth,
      height,
    })

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    })

    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    })

    chart.priceScale('volume').applyOptions({
      scaleMargins: { top: 0.85, bottom: 0 },
    })

    chart.subscribeCrosshairMove((param) => {
      if (!param.time || !param.point) {
        onCrosshairMoveRef.current?.(null)
        onCandleHoverRef.current?.(null)
        return
      }
      const candleData = param.seriesData.get(candleSeries)
      if (candleData) {
        const close = candleData['close']
        if (typeof close === 'number') {
          onCrosshairMoveRef.current?.({ time: param.time as number, price: close })
        }
        const open = candleData['open']
        const high = candleData['high']
        const low = candleData['low']
        const vol = candleData['volume']
        if (typeof open === 'number' && typeof high === 'number' && typeof low === 'number') {
          onCandleHoverRef.current?.({
            time: param.time as number,
            open, high, low, close: typeof close === 'number' ? close : 0,
            volume: typeof vol === 'number' ? vol : undefined,
          })
        }
      }
    })

    chartRef.current = chart
    candleSeriesRef.current = candleSeries
    volumeSeriesRef.current = volumeSeries
    indicatorSeriesRef.current.clear()
    priceLinesRef.current = []
    markersRef.current = null
    prevMarkersKeyRef.current = ''

    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width } = entry.contentRect
        chart.applyOptions({ width })
      }
    })
    resizeObserver.observe(containerRef.current)

    chart.timeScale().subscribeVisibleLogicalRangeChange((range) => {
      if (!range) return
      const timeRange = chart.timeScale().getVisibleRange()
      if (timeRange && onVisibleRangeChangeRef.current) {
        onVisibleRangeChangeRef.current({
          from: timeRange.from as number,
          to: timeRange.to as number,
        })
      }
    })

    return () => {
      resizeObserver.disconnect()
      chart.remove()
    }
  }, [height])

  useEffect(() => {
    const cleanup = initChart()
    return () => cleanup?.()
  }, [initChart])

  // Candle data + trade markers
  useEffect(() => {
    if (!candleSeriesRef.current || !volumeSeriesRef.current || candles.length === 0) return

    const prevCount = prevCandleCountRef.current
    const isInitialLoad = prevCount === 0
    const hasMoreHistory = candles.length > prevCount && prevCount > 0 &&
      candles[0] !== undefined
    prevCandleCountRef.current = candles.length

    if (isInitialLoad || candles.length < prevCount) {
      candleSeriesRef.current.setData(candles.map((c) => ({
        time: c.time, open: c.open, high: c.high, low: c.low, close: c.close,
      })))
      volumeSeriesRef.current.setData(candles
        .filter((c) => c.volume != null && c.volume > 0)
        .map((c) => ({
          time: c.time,
          value: c.volume!,
          color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
        })))
      const ts = chartRef.current?.timeScale()
      if (ts && candles.length > 80) {
        const visibleStart = candles[candles.length - 80]
        const visibleEnd = candles[candles.length - 1]
        if (visibleStart && visibleEnd) {
          ts.setVisibleRange({ from: visibleStart.time, to: visibleEnd.time })
        }
      } else {
        ts?.fitContent()
      }
    } else if (hasMoreHistory) {
      candleSeriesRef.current.setData(candles.map((c) => ({
        time: c.time, open: c.open, high: c.high, low: c.low, close: c.close,
      })))
      volumeSeriesRef.current.setData(candles
        .filter((c) => c.volume != null && c.volume > 0)
        .map((c) => ({
          time: c.time,
          value: c.volume!,
          color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
        })))
    } else {
      const lastCandle = candles[candles.length - 1]
      if (!lastCandle) return
      candleSeriesRef.current.update({
        time: lastCandle.time, open: lastCandle.open, high: lastCandle.high, low: lastCandle.low, close: lastCandle.close,
      })
      if (lastCandle.volume != null && lastCandle.volume > 0) {
        volumeSeriesRef.current.update({
          time: lastCandle.time,
          value: lastCandle.volume,
          color: lastCandle.close >= lastCandle.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
        })
      }
    }

    if (tradeMarkers && tradeMarkers.length > 0) {
      const markers = tradeMarkers
        .sort((a, b) => (a.time as number) - (b.time as number))
        .map((t) => {
          const action = t.positionAction
          const actionLabel = action === 'open' ? ' Open' : action === 'close' ? ' Close' : action === 'add' ? ' Add' : action === 'reduce' ? ' Reduce' : ''
          const isOpen = action === 'open'
          const isClose = action === 'close'
          return {
            time: t.time,
            position: t.side === 'BUY' ? 'belowBar' as const : 'aboveBar' as const,
            color: isOpen ? '#3b82f6' : isClose ? '#f97316' : t.side === 'BUY' ? '#22c55e' : '#ef4444',
            shape: isOpen ? 'circle' as const : isClose ? 'square' as const : t.side === 'BUY' ? 'arrowUp' as const : 'arrowDown' as const,
            text: `${t.side === 'BUY' ? '▲' : '▼'}${actionLabel} ${t.quantity}`,
          }
        })
      const markersKey = markers.map((m) => `${m.time}-${m.shape}-${m.text}`).join('|')
      if (markersKey !== prevMarkersKeyRef.current) {
        prevMarkersKeyRef.current = markersKey
        if (markersRef.current) {
          markersRef.current.setMarkers(markers)
        } else {
          markersRef.current = createSeriesMarkers(candleSeriesRef.current, markers)
        }
      }
    } else if (markersRef.current) {
      markersRef.current.setMarkers([])
      prevMarkersKeyRef.current = ''
    }
  }, [candles, tradeMarkers])

  // Order levels + grid lines as price lines
  useEffect(() => {
    if (!candleSeriesRef.current) return

    for (const pl of priceLinesRef.current) {
      candleSeriesRef.current.removePriceLine(pl)
    }
    priceLinesRef.current = []

    const allLines: Array<{ price: number; color: string; lineStyle: number; title: string }> = []

    for (const o of orderLevels ?? []) {
      allLines.push({
        price: o.price,
        color: o.side === 'BUY' ? 'rgba(34, 197, 94, 0.5)' : 'rgba(239, 68, 68, 0.5)',
        lineStyle: 2,
        title: `${o.side} ${o.quantity}`,
      })
    }

    for (const g of gridLines ?? []) {
      allLines.push({
        price: g.price,
        color: g.color,
        lineStyle: 3,
        title: g.label,
      })
    }

    if (allLines.length === 0) return

    for (const line of allLines) {
      const pl = candleSeriesRef.current.createPriceLine({
        price: line.price,
        color: line.color,
        lineWidth: 1,
        lineStyle: line.lineStyle as LineStyle,
        axisLabelVisible: true,
        title: line.title,
      })
      priceLinesRef.current.push(pl)
    }
  }, [orderLevels, gridLines])

  // Indicator lines as overlay series
  useEffect(() => {
    if (!chartRef.current) return

    for (const [id, series] of indicatorSeriesRef.current) {
      const stillExists = indicatorLines?.some((il) => il.id === id)
      if (!stillExists) {
        try { chartRef.current.removeSeries(series) } catch { /* ignore */ }
        indicatorSeriesRef.current.delete(id)
      }
    }

    if (!indicatorLines || indicatorLines.length === 0) return

    for (const il of indicatorLines) {
      if (il.data.length === 0) continue
      let series = indicatorSeriesRef.current.get(il.id)
      if (!series) {
        const opts: Record<string, unknown> = {
          color: il.color,
          lineWidth: (il.lineWidth ?? 1) as 1 | 2 | 3 | 4,
          lineStyle: il.lineStyle as LineStyle,
          priceLineVisible: false,
          lastValueVisible: true,
          crosshairMarkerVisible: false,
        }
        if (il.priceScaleId) {
          opts.priceScaleId = il.priceScaleId
        }
        series = chartRef.current.addSeries(LineSeries, opts)
        if (il.priceScaleId && il.scaleMargins) {
          chartRef.current.priceScale(il.priceScaleId).applyOptions({
            scaleMargins: il.scaleMargins,
          })
        }
        indicatorSeriesRef.current.set(il.id, series)
      }
      series.setData(il.data.map((d) => ({ time: d.time, value: d.value })))
    }
  }, [indicatorLines])

  // Click handler for trade marker tooltips
  useEffect(() => {
    if (!chartRef.current || !tradeMarkers || tradeMarkers.length === 0) return

    const handler = (param: { point?: { x: number; y: number }; time?: Time }) => {
      if (!param.point || !param.time) {
        setTooltip(null)
        return
      }
      const clickTime = param.time as number
      const closest = [...tradeMarkers]
        .map((t) => ({ marker: t, timeDist: Math.abs((t.time as number) - clickTime) }))
        .filter((t) => t.timeDist < 300)
        .sort((a, b) => a.timeDist - b.timeDist)[0]

      if (closest) {
        setTooltip({
          x: param.point.x,
          y: closest.marker.side === 'BUY' ? param.point.y - 30 : param.point.y + 10,
          data: closest.marker,
        })
      } else {
        setTooltip(null)
      }
    }

    chartRef.current.subscribeClick(handler)
    return () => { chartRef.current?.unsubscribeClick(handler) }
  }, [tradeMarkers])

  return (
    <div className="relative">
      <div ref={containerRef} className="rounded-lg overflow-hidden" style={{ height }} />
      {tooltip && (
        <TradeTooltip
          x={tooltip.x}
          y={tooltip.y}
          data={tooltip.data}
          onClose={() => setTooltip(null)}
        />
      )}
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-card/80">
          <span className="text-sm text-muted-foreground">Loading chart...</span>
        </div>
      )}
      {!isLoading && candles.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-sm text-muted-foreground">No candlestick data available</span>
        </div>
      )}
    </div>
  )
}

function TradeTooltip({ x, y, data, onClose }: {
  x: number
  y: number
  data: TradeMarker
  onClose: () => void
}) {
  return (
    <div
      className="absolute z-50 w-52 rounded-lg border bg-card p-2.5 shadow-lg text-xs"
      style={{ left: Math.min(x, 300), top: Math.max(y, 10) }}
      onClick={(e) => e.stopPropagation()}
    >
      <button
        onClick={onClose}
        className="absolute top-1 right-1.5 text-muted-foreground hover:text-foreground text-sm leading-none"
      >
        ×
      </button>
      <div className="flex items-center gap-1.5 mb-1.5">
        <span className={`inline-block h-2 w-2 rounded-full ${data.side === 'BUY' ? 'bg-trade-up' : 'bg-trade-down'}`} />
        <span className={`font-semibold ${data.side === 'BUY' ? 'text-trade-up' : 'text-trade-down'}`}>
          {data.side}
        </span>
        {data.isMaker && (
          <span className="rounded bg-muted px-1 py-0.5 text-[10px] text-muted-foreground">Maker</span>
        )}
      </div>
      <div className="space-y-0.5 font-mono text-muted-foreground">
        <div className="flex justify-between">
          <span>Price</span>
          <span className="text-foreground">{data.price}</span>
        </div>
        <div className="flex justify-between">
          <span>Qty</span>
          <span className="text-foreground">{data.quantity}</span>
        </div>
        {data.fee && (
          <div className="flex justify-between">
            <span>Fee</span>
            <span className="text-foreground">{data.fee} {data.feeCurrency}</span>
          </div>
        )}
      </div>
    </div>
  )
}
