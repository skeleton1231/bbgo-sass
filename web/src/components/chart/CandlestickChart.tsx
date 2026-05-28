'use client'

import { useEffect, useRef, useCallback } from 'react'
import {
  createChart,
  createSeriesMarkers,
  CandlestickSeries,
  HistogramSeries,
  type IChartApi,
  type ISeriesApi,
  type SeriesType,
  type Time,
  type DeepPartial,
  type ChartOptions,
  type ISeriesMarkersPluginApi,
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
}

export interface OrderLevel {
  price: number
  side: 'BUY' | 'SELL'
  quantity: string
}

interface CandlestickChartProps {
  candles: KlineCandle[]
  tradeMarkers?: TradeMarker[]
  orderLevels?: OrderLevel[]
  height?: number
  isLoading?: boolean
  dataKey?: string
  onVisibleTimeRangeChange?: (range: { from: number; to: number } | null) => void
}

function getChartTheme(): DeepPartial<ChartOptions> {
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  return {
    layout: {
      background: { type: ColorType.Solid, color: isDark ? '#0a0a0a' : '#ffffff' },
      textColor: isDark ? '#a1a1aa' : '#71717a',
    },
    grid: {
      vertLines: { color: isDark ? '#1e1e1e' : '#f4f4f5' },
      horzLines: { color: isDark ? '#1e1e1e' : '#f4f4f5' },
    },
    crosshair: {
      vertLine: { color: '#71717a', width: 1, style: 2 },
      horzLine: { color: '#71717a', width: 1, style: 2 },
    },
    rightPriceScale: {
      borderColor: isDark ? '#27272a' : '#e4e4e7',
    },
    timeScale: {
      borderColor: isDark ? '#27272a' : '#e4e4e7',
      timeVisible: true,
      secondsVisible: false,
    },
  }
}

export function CandlestickChart({
  candles,
  tradeMarkers,
  orderLevels,
  height = 400,
  isLoading,
  dataKey,
  onVisibleTimeRangeChange,
}: CandlestickChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const volumeSeriesRef = useRef<ISeriesApi<SeriesType> | null>(null)
  const markersRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)
  const priceLinesRef = useRef<ReturnType<ISeriesApi<SeriesType>['createPriceLine']>[]>([])
  const prevCandleCountRef = useRef(0)
  const prevDataKeyRef = useRef(dataKey)
  const onVisibleRangeChangeRef = useRef(onVisibleTimeRangeChange)
  onVisibleRangeChangeRef.current = onVisibleTimeRangeChange

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

    chartRef.current = chart
    candleSeriesRef.current = candleSeries
    volumeSeriesRef.current = volumeSeries
    priceLinesRef.current = []
    markersRef.current = null

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

  useEffect(() => {
    if (!candleSeriesRef.current || !volumeSeriesRef.current || candles.length === 0) return

    const prevCount = prevCandleCountRef.current
    const isInitialLoad = prevCount === 0
    const hasMoreHistory = candles.length > prevCount && prevCount > 0 &&
      candles[0] !== undefined
    prevCandleCountRef.current = candles.length

    if (isInitialLoad || candles.length < prevCount) {
      // Full dataset replacement: initial load or interval change
      const candleData = candles.map((c) => ({
        time: c.time,
        open: c.open,
        high: c.high,
        low: c.low,
        close: c.close,
      }))
      candleSeriesRef.current.setData(candleData)

      const volumeData = candles
        .filter((c) => c.volume != null && c.volume > 0)
        .map((c) => ({
          time: c.time,
          value: c.volume!,
          color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
        }))
      volumeSeriesRef.current.setData(volumeData)
      chartRef.current?.timeScale().fitContent()
    } else if (hasMoreHistory) {
      // Prepending older candles — replace data but don't fit
      const candleData = candles.map((c) => ({
        time: c.time,
        open: c.open,
        high: c.high,
        low: c.low,
        close: c.close,
      }))
      candleSeriesRef.current.setData(candleData)

      const volumeData = candles
        .filter((c) => c.volume != null && c.volume > 0)
        .map((c) => ({
          time: c.time,
          value: c.volume!,
          color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
        }))
      volumeSeriesRef.current.setData(volumeData)
      // Don't call fitContent — preserve user's scroll position
    } else {
      // Incremental update — only update last candle (WS tick)
      const lastCandle = candles[candles.length - 1]
      if (!lastCandle) return
      candleSeriesRef.current.update({
        time: lastCandle.time,
        open: lastCandle.open,
        high: lastCandle.high,
        low: lastCandle.low,
        close: lastCandle.close,
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
        .map((t) => ({
          time: t.time,
          position: t.side === 'BUY' ? 'belowBar' as const : 'aboveBar' as const,
          color: t.side === 'BUY' ? '#22c55e' : '#ef4444',
          shape: t.side === 'BUY' ? 'arrowUp' as const : 'arrowDown' as const,
          text: `${t.side} ${t.quantity}`,
        }))
      markersRef.current = createSeriesMarkers(candleSeriesRef.current, markers)
    }
  }, [candles, tradeMarkers])

  useEffect(() => {
    if (!candleSeriesRef.current || !orderLevels) return

    for (const pl of priceLinesRef.current) {
      candleSeriesRef.current.removePriceLine(pl)
    }
    priceLinesRef.current = []

    if (orderLevels.length === 0) return

    for (const o of orderLevels) {
      const pl = candleSeriesRef.current.createPriceLine({
        price: o.price,
        color: o.side === 'BUY' ? 'rgba(34, 197, 94, 0.5)' : 'rgba(239, 68, 68, 0.5)',
        lineWidth: 1,
        lineStyle: 2,
        axisLabelVisible: true,
        title: `${o.side} ${o.quantity}`,
      })
      priceLinesRef.current.push(pl)
    }
  }, [orderLevels])

  return (
    <div className="relative">
      <div ref={containerRef} className="rounded-lg overflow-hidden" style={{ height }} />
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
