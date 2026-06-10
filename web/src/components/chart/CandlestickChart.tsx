'use client'

import { useEffect, useRef, useCallback, useState } from 'react'
import { useTranslations } from 'next-intl'
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
    | 'openLong' | 'closeLong' | 'openShort' | 'closeShort'
    | 'addLong' | 'reduceLong' | 'addShort' | 'reduceShort'
    | 'flipLongToShort' | 'flipShortToLong'
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
  liquidationPrice?: number
}

type MarkerAction = TradeMarker['positionAction']

const MARKER_COLORS = {
  openLong: '#3b82f6',
  closeLong: '#f97316',
  openShort: '#ef4444',
  closeShort: '#22c55e',
  addLong: '#60a5fa',
  reduceLong: '#fb923c',
  addShort: '#f87171',
  reduceShort: '#4ade80',
  flipLongToShort: '#a855f7',
  flipShortToLong: '#a855f7',
  buy: '#22c55e',
  sell: '#ef4444',
} as const

interface MarkerVisual {
  position: 'belowBar' | 'aboveBar'
  color: string
  shape: 'circle' | 'square' | 'arrowUp' | 'arrowDown'
  icon: string
  label: string
}

function getMarkerInfo(action: MarkerAction, side: 'BUY' | 'SELL'): MarkerVisual {
  const c = MARKER_COLORS
  switch (action) {
    case 'openLong':
      return { position: 'belowBar', color: c.openLong, shape: 'circle', icon: '▲', label: 'Open Long' }
    case 'closeLong':
      return { position: 'aboveBar', color: c.closeLong, shape: 'square', icon: '▼', label: 'Close Long' }
    case 'openShort':
      return { position: 'aboveBar', color: c.openShort, shape: 'circle', icon: '▼', label: 'Open Short' }
    case 'closeShort':
      return { position: 'belowBar', color: c.closeShort, shape: 'square', icon: '▲', label: 'Close Short' }
    case 'addLong':
      return { position: 'belowBar', color: c.addLong, shape: 'arrowUp', icon: '▲', label: 'Add Long' }
    case 'reduceLong':
      return { position: 'aboveBar', color: c.reduceLong, shape: 'arrowDown', icon: '▼', label: 'Reduce Long' }
    case 'addShort':
      return { position: 'aboveBar', color: c.addShort, shape: 'arrowDown', icon: '▼', label: 'Add Short' }
    case 'reduceShort':
      return { position: 'belowBar', color: c.reduceShort, shape: 'arrowUp', icon: '▲', label: 'Reduce Short' }
    case 'flipLongToShort':
      return { position: 'aboveBar', color: c.flipLongToShort, shape: 'square', icon: '▼▲', label: 'Flip → Short' }
    case 'flipShortToLong':
      return { position: 'belowBar', color: c.flipShortToLong, shape: 'square', icon: '▲▼', label: 'Flip → Long' }
    case 'open':
      return { position: 'belowBar', color: c.openLong, shape: 'circle', icon: '▲', label: 'Open' }
    case 'close':
      return { position: 'aboveBar', color: c.closeLong, shape: 'square', icon: '▼', label: 'Close' }
    case 'add':
      return { position: 'belowBar', color: c.addLong, shape: 'arrowUp', icon: '▲', label: 'Add' }
    case 'reduce':
      return { position: 'aboveBar', color: c.reduceLong, shape: 'arrowDown', icon: '▼', label: 'Reduce' }
    default:
      return side === 'BUY'
        ? { position: 'belowBar', color: c.buy, shape: 'arrowUp', icon: '▲', label: '' }
        : { position: 'aboveBar', color: c.sell, shape: 'arrowDown', icon: '▼', label: '' }
  }
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
  liquidationPrice,
}: CandlestickChartProps) {
  const t = useTranslations('Bots')
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const volumeSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const indicatorSeriesRef = useRef<Map<string, ISeriesApi<SeriesType>>>(new Map())
  const markersRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)
  const priceLinesRef = useRef<ReturnType<ISeriesApi<SeriesType>['createPriceLine']>[]>([])
  const prevCandleCountRef = useRef(0)
  const prevDataKeyRef = useRef(dataKey)
  const hadDataRef = useRef(false)
  const isInitialLoadRef = useRef(true)
  const onVisibleRangeChangeRef = useRef(onVisibleTimeRangeChange)
  const onCrosshairMoveRef = useRef(onCrosshairMove)
  const [tooltip, setTooltip] = useState<{ x: number; y: number; data: TradeMarker } | null>(null)
  const onCandleHoverRef = useRef(onCandleHover)
  const prevMarkersKeyRef = useRef('')

  useEffect(() => {
    onVisibleRangeChangeRef.current = onVisibleTimeRangeChange
    onCrosshairMoveRef.current = onCrosshairMove
    onCandleHoverRef.current = onCandleHover
  }, [onVisibleTimeRangeChange, onCrosshairMove, onCandleHover])

  useEffect(() => {
    if (prevDataKeyRef.current !== dataKey) {
      prevDataKeyRef.current = dataKey
      prevCandleCountRef.current = 0
      hadDataRef.current = false
      isInitialLoadRef.current = true
    }
  }, [dataKey])

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
    isInitialLoadRef.current = true

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
    if (!candleSeriesRef.current || !volumeSeriesRef.current) return

    if (candles.length === 0) {
      prevCandleCountRef.current = 0
      hadDataRef.current = false
      return
    }

    const prevCount = prevCandleCountRef.current
    prevCandleCountRef.current = candles.length

    const candleData = candles.map((c) => ({
      time: c.time, open: c.open, high: c.high, low: c.low, close: c.close,
    }))
    const volumeData = candles
      .filter((c) => c.volume != null && c.volume > 0)
      .map((c) => ({
        time: c.time,
        value: c.volume!,
        color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
      }))

    const useSetData = !hadDataRef.current || prevCount === 0 || candles.length !== prevCount
    hadDataRef.current = true

    if (useSetData) {
      candleSeriesRef.current.setData(candleData)
      volumeSeriesRef.current.setData(volumeData)
      const ts = chartRef.current?.timeScale()
      if (isInitialLoadRef.current) {
        isInitialLoadRef.current = false
        if (ts && candles.length > 80) {
          const visibleStart = candles[candles.length - 80]
          const visibleEnd = candles[candles.length - 1]
          if (visibleStart && visibleEnd) {
            ts.setVisibleRange({ from: visibleStart.time, to: visibleEnd.time })
          }
        } else {
          ts?.fitContent()
        }
      }
    } else {
      const lastCandle = candles[candles.length - 1]
      if (!lastCandle) return
      try {
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
      } catch {
        candleSeriesRef.current.setData(candleData)
        volumeSeriesRef.current.setData(volumeData)
      }
    }

    if (tradeMarkers && tradeMarkers.length > 0) {
      const candleTimes = candles.map((c) => c.time as number)
      const markers = tradeMarkers
        .sort((a, b) => (a.time as number) - (b.time as number))
        .map((t) => {
          const a = t.positionAction
          const info = getMarkerInfo(a, t.side)
          const snappedTime = snapToNearestCandle(t.time as number, candleTimes)
          return {
            time: (snappedTime ?? t.time) as Time,
            position: info.position,
            color: info.color,
            shape: info.shape,
            text: `${info.icon} ${info.label} ${t.quantity}`,
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

    if (liquidationPrice && liquidationPrice > 0) {
      allLines.push({
        price: liquidationPrice,
        color: 'rgba(239, 68, 68, 0.8)',
        lineStyle: 2,
        title: '⚠ Liq',
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
  }, [orderLevels, gridLines, liquidationPrice])

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
          <span className="text-sm text-muted-foreground">{t('loading')}</span>
        </div>
      )}
      {!isLoading && candles.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-sm text-muted-foreground">{t('noCandleData')}</span>
        </div>
      )}
    </div>
  )
}

function snapToNearestCandle(target: number, sortedCandleTimes: number[]): number | null {
  if (sortedCandleTimes.length === 0) return null
  if (sortedCandleTimes.length === 1) {
    const diff = Math.abs(target - sortedCandleTimes[0]!)
    return diff <= 7200 ? sortedCandleTimes[0]! : null
  }

  let lo = 0
  let hi = sortedCandleTimes.length - 1
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (sortedCandleTimes[mid]! < target) lo = mid + 1
    else hi = mid
  }

  const prev = lo > 0 ? sortedCandleTimes[lo - 1]! : null
  const next = sortedCandleTimes[lo]!

  let nearest: number
  if (!prev) {
    nearest = next
  } else {
    nearest = (target - prev) <= (next - target) ? prev : next
  }

  return Math.abs(target - nearest) <= 7200 ? nearest : null
}

function TradeTooltip({ x, y, data, onClose }: {
  x: number
  y: number
  data: TradeMarker
  onClose: () => void
}) {
  const t = useTranslations('Bots')
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
          <span className="rounded bg-muted px-1 py-0.5 text-[10px] text-muted-foreground">{t('tradeTags.maker')}</span>
        )}
      </div>
      <div className="space-y-0.5 font-mono text-muted-foreground">
        <div className="flex justify-between">
          <span>{t('tooltipPrice')}</span>
          <span className="text-foreground">{data.price}</span>
        </div>
        <div className="flex justify-between">
          <span>{t('tooltipQty')}</span>
          <span className="text-foreground">{data.quantity}</span>
        </div>
        {data.fee && (
          <div className="flex justify-between">
            <span>{t('tooltipFee')}</span>
            <span className="text-foreground">{data.fee} {data.feeCurrency}</span>
          </div>
        )}
      </div>
    </div>
  )
}
