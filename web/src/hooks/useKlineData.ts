'use client'

import { useState, useCallback, useMemo, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
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
  mode?: 'live' | 'paper'
  session?: string // e.g., "binance_futures" to use futures market data
  enabled?: boolean
}

function parseKlineRaw(k: { time: string | number; open: string; high: string; low: string; close: string; volume: string }): KlineCandle {
  return {
    time: (typeof k.time === 'string' ? Math.floor(new Date(k.time).getTime() / 1000) : Math.floor(Number(k.time) / 1000)) as Time,
    open: parseFloat(k.open),
    high: parseFloat(k.high),
    low: parseFloat(k.low),
    close: parseFloat(k.close),
    volume: parseFloat(k.volume || '0'),
  }
}

export function useKlineData({ userId, exchange, symbol, interval, mode, session, enabled = true }: UseKlineDataOptions) {
  const queryClient = useQueryClient()
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  const loadingMoreRef = useRef(false)
  const wsCandlesRef = useRef<Map<string, KlineCandle>>(new Map())
  const earliestTimeRef = useRef<number | null>(null)

  const queryKey = useMemo(() => ['klines', exchange, symbol, interval, mode, session] as const, [exchange, symbol, interval, mode, session])

  const { data: candles = [], isLoading, error, refetch } = useQuery<KlineCandle[]>({
    queryKey,
    queryFn: async () => {
      if (!exchange || !symbol) return []
      const params = new URLSearchParams({
        symbol,
        interval: interval || '1h',
        limit: '200',
      })
      if (mode) params.set('mode', mode)
      if (session) params.set('session', session)
      const res = await fetch(`/api/manager/markets/${encodeURIComponent(exchange)}/klines?${params}`)
      if (!res.ok) throw new Error(`Failed to fetch klines: ${res.status}`)
      const data = await res.json()
      const klines: KlineCandle[] = (data.klines || []).map(parseKlineRaw)

      wsCandlesRef.current.clear()
      for (const k of klines) {
        wsCandlesRef.current.set(String(k.time), k)
      }
      if (klines.length > 0 && klines[0]) {
        earliestTimeRef.current = klines[0].time as number
      }
      return klines
    },
    enabled: enabled && !!exchange && !!symbol,
    staleTime: 60_000,
  })

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
      if (mode) params.set('mode', mode)
      if (session) params.set('session', session)
      const res = await fetch(`/api/manager/markets/${encodeURIComponent(exchange)}/klines?${params}`)
      if (!res.ok) return

      const data = await res.json()
      const olderKlines: KlineCandle[] = (data.klines || []).map(parseKlineRaw)
      if (olderKlines.length === 0) return

      for (const k of olderKlines) {
        wsCandlesRef.current.set(String(k.time), k)
      }
      if (olderKlines[0]) {
        earliestTimeRef.current = olderKlines[0].time as number
      }

      const sorted = Array.from(wsCandlesRef.current.values())
        .sort((a, b) => (a.time as number) - (b.time as number))
      queryClient.setQueryData(queryKey, sorted)
    } finally {
      loadingMoreRef.current = false
      setIsLoadingMore(false)
    }
  }, [exchange, symbol, interval, enabled, mode, session, queryClient, queryKey])

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
      queryClient.setQueryData(queryKey, sorted)
    } else {
      queryClient.setQueryData(queryKey, (prev: KlineCandle[] | undefined) => {
        if (!prev) return prev
        const idx = prev.findIndex(c => c.time === time)
        if (idx === -1) return prev
        const next = [...prev]
        next[idx] = newCandle
        return next
      })
    }
  }, [exchange, symbol, queryClient, queryKey])

  useMarketData({
    userId,
    mode,
    enabled: enabled && !!exchange && !!symbol,
    onMessage: handleWSMessage,
  })

  return {
    candles,
    isLoading,
    isLoadingMore,
    error: error?.message ?? null,
    refetch,
    loadEarlierKlines,
  }
}
