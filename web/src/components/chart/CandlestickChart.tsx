'use client'

import { useEffect, useRef, useCallback } from 'react'
import {
  createChart,
  type IChartApi,
  type ISeriesApi,
  type CandlestickData,
  type Time,
  type HistogramData,
  type DeepPartial,
  type ChartOptions,
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
}: CandlestickChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const volumeSeriesRef = useRef<ISeriesApi<'Histogram'> | null>(null)
  const priceLinesRef = useRef<ReturnType<ISeriesApi<'Candlestick'>['createPriceLine']>[]>([])

  const initChart = useCallback(() => {
    if (!containerRef.current) return

    if (chartRef.current) {
      chartRef.current.remove()
      chartRef.current = null
    }

    const chart = createChart(containerRef.current, {
      ...getChartTheme(),
      width: containerRef.current.clientWidth,
      height,
    })

    const candleSeries = chart.addCandlestickSeries({
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    })

    const volumeSeries = chart.addHistogramSeries({
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

    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width } = entry.contentRect
        chart.applyOptions({ width })
      }
    })
    resizeObserver.observe(containerRef.current)

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

    const candleData: CandlestickData[] = candles.map((c) => ({
      time: c.time,
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
    }))
    candleSeriesRef.current.setData(candleData)

    const volumeData: HistogramData[] = candles
      .filter((c) => c.volume != null && c.volume > 0)
      .map((c) => ({
        time: c.time,
        value: c.volume!,
        color: c.close >= c.open ? 'rgba(34, 197, 94, 0.3)' : 'rgba(239, 68, 68, 0.3)',
      }))
    volumeSeriesRef.current.setData(volumeData)

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
      candleSeriesRef.current.setMarkers(markers)
    }

    chartRef.current?.timeScale().fitContent()
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center rounded-lg border bg-card" style={{ height }}>
        <span className="text-sm text-muted-foreground">Loading chart...</span>
      </div>
    )
  }

  return (
    <div className="relative">
      <div ref={containerRef} className="rounded-lg overflow-hidden" />
      {candles.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-sm text-muted-foreground">No candlestick data available</span>
        </div>
      )}
    </div>
  )
}
