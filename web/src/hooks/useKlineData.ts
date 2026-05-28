'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import type { Time } from 'lightweight-charts'
import { useMarketData } from '@/lib/bbgo/useWebSocket'
import type { KlineCandle } from '@/components/chart/CandlestickChart'

interface KlineUpdate {
  open: string
  high: string
  low: string
  close: string
  volume: string
  startTime: string
  closed: boolean
}

interface UseKlineDataOptions {
  userId: string
  exchange: string
  symbol: string
  interval?: string
  enabled?: boolean
}

export function useKlineData({ userId, exchange, symbol, interval, enabled = true }: UseKlineDataOptions) {
  const [candles, setCandles] = useState<KlineCandle[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const wsCandlesRef = useRef<Map<string, KlineCandle>>(new Map())
  const earliestTimeRef = useRef<number | null>(null)
  const loadingMoreRef = useRef(false)

  const fetchHistoricalKlines = useCallback(async () => {
    if (!exchange || !symbol || !enabled) return

    setCandles([])
    setIsLoading(true)
    setError(null)
    earliestTimeRef.current = null

    try {
      const params = new URLSearchParams({
        symbol,
        interval: interval || '1h',
        limit: '200',
      })
      const res = await fetch(`/api/manager/markets/${encodeURIComponent(exchange)}/klines?${params}`)
      if (!res.ok) {
        throw new Error(`Failed to fetch klines: ${res.status}`)
      }
      const data = await res.json()
      const klines: KlineCandle[] = (data.klines || []).map((k: { time: string | number; open: string; high: string; low: string; close: string; volume: string }) => ({
        time: (typeof k.time === 'string' ? Math.floor(new Date(k.time).getTime() / 1000) : Math.floor(Number(k.time) / 1000)) as Time,
        open: parseFloat(k.open),
        high: parseFloat(k.high),
        low: parseFloat(k.low),
        close: parseFloat(k.close),
        volume: parseFloat(k.volume || '0'),
      }))

      setCandles(klines)
      wsCandlesRef.current.clear()
      for (const k of klines) {
        wsCandlesRef.current.set(String(k.time), k)
      }
      if (klines.length > 0 && klines[0]) {
        earliestTimeRef.current = klines[0].time as number
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load kline data')
    } finally {
      setIsLoading(false)
    }
  }, [exchange, symbol, interval, enabled])

  const loadEarlierKlines = useCallback(async () => {
    if (!exchange || !symbol || !enabled || loadingMoreRef.current || !earliestTimeRef.current) return

    loadingMoreRef.current = true
    setIsLoadingMore(true)

    try {
      const endTime = (earliestTimeRef.current - 1) * 1000
      const params = new URLSearchParams({
        symbol,
        interval: interval || '1h',
        limit: '200',
        end_time: String(endTime),
      })
      const res = await fetch(`/api/manager/markets/${encodeURIComponent(exchange)}/klines?${params}`)
      if (!res.ok) return

      const data = await res.json()
      const olderKlines: KlineCandle[] = (data.klines || []).map((k: { time: string | number; open: string; high: string; low: string; close: string; volume: string }) => ({
        time: (typeof k.time === 'string' ? Math.floor(new Date(k.time).getTime() / 1000) : Math.floor(Number(k.time) / 1000)) as Time,
        open: parseFloat(k.open),
        high: parseFloat(k.high),
        low: parseFloat(k.low),
        close: parseFloat(k.close),
        volume: parseFloat(k.volume || '0'),
      }))

      if (olderKlines.length === 0) return

      for (const k of olderKlines) {
        wsCandlesRef.current.set(String(k.time), k)
      }
      if (olderKlines[0]) {
        earliestTimeRef.current = olderKlines[0].time as number
      }

      const sorted = Array.from(wsCandlesRef.current.values())
        .sort((a, b) => (a.time as number) - (b.time as number))
      setCandles(sorted)
    } finally {
      loadingMoreRef.current = false
      setIsLoadingMore(false)
    }
  }, [exchange, symbol, interval, enabled])

  useEffect(() => {
    fetchHistoricalKlines()
  }, [fetchHistoricalKlines])

  const handleWSMessage = useCallback((msg: { type: string; data: { exchange?: string; symbol?: string; channel?: string; kline?: KlineUpdate } }) => {
    if (msg.type !== 'market') return
    if (!msg.data.kline) return
    if (msg.data.exchange !== exchange || msg.data.symbol !== symbol) return

    const kl = msg.data.kline
    const time = Math.floor(new Date(kl.startTime).getTime() / 1000) as Time
    const newCandle: KlineCandle = {
      time,
      open: parseFloat(kl.open),
      high: parseFloat(kl.high),
      low: parseFloat(kl.low),
      close: parseFloat(kl.close),
      volume: parseFloat(kl.volume || '0'),
    }

    const isNew = !wsCandlesRef.current.has(String(time))
    wsCandlesRef.current.set(String(time), newCandle)

    if (isNew) {
      const sorted = Array.from(wsCandlesRef.current.values())
        .sort((a, b) => (a.time as number) - (b.time as number))
      setCandles(sorted)
    } else {
      setCandles(prev => {
        const idx = prev.findIndex(c => c.time === time)
        if (idx === -1) return prev
        const next = [...prev]
        next[idx] = newCandle
        return next
      })
    }
  }, [exchange, symbol])

  useMarketData({
    userId,
    enabled: enabled && !!exchange && !!symbol,
    onMessage: handleWSMessage,
  })

  return { candles, isLoading, isLoadingMore, error, refetch: fetchHistoricalKlines, loadEarlierKlines }
}
