'use client'

import { useEffect, useRef, useState, useCallback } from 'react'

export interface MarketDataMessage {
  type: 'market' | 'userData'
  data: {
    exchange?: string
    symbol?: string
    channel?: string
    event?: string
    depth?: { bids: Array<{ price: string; volume: string }>; asks: Array<{ price: string; volume: string }> }
    kline?: { open: string; high: string; low: string; close: string; volume: string; startTime: string; closed: boolean }
    ticker?: { open: number; high: number; low: number; close: number; volume: number }
    trades?: Array<{ id: string; price: string; quantity: string; createdAt: string; side: string }>
    balances?: Array<{ currency: string; available: string; locked: string }>
    orders?: Array<{ id: string; symbol: string; side: string; price: string; quantity: string; executedQuantity: string; status: string }>
  }
}

interface UseWebSocketOptions {
  userId: string
  mode?: 'live' | 'paper'
  enabled?: boolean
  onMessage?: (msg: MarketDataMessage) => void
}

export function useMarketData({ userId, mode, enabled = true, onMessage }: UseWebSocketOptions) {
  const [lastMessage, setLastMessage] = useState<MarketDataMessage | null>(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const onMessageRef = useRef(onMessage)
  const connectRef = useRef<() => void>(() => {})
  const mountedRef = useRef(true)

  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  const connect = useCallback(async () => {
    if (!enabled || !userId || !mountedRef.current) return

    try {
      const params = new URLSearchParams()
      if (mode) params.set('mode', mode)
      const res = await fetch(`/api/ws-url?${params}`)
      if (!res.ok) return
      const { wsUrl } = await res.json()
      if (!wsUrl) return

      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => setConnected(true)
      ws.onclose = () => {
        setConnected(false)
        wsRef.current = null
        reconnectRef.current = setTimeout(() => connectRef.current(), 5_000)
      }
      ws.onerror = () => ws.close()
      ws.onmessage = (e) => {
        try {
          const msg: MarketDataMessage = JSON.parse(e.data)
          setLastMessage(msg)
          onMessageRef.current?.(msg)
        } catch { /* ignore malformed messages */ }
      }
    } catch {
      reconnectRef.current = setTimeout(() => connectRef.current(), 5_000)
    }
  }, [userId, mode, enabled])

  useEffect(() => {
    connectRef.current = connect
  }, [connect])

  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      clearTimeout(reconnectRef.current)
      wsRef.current?.close()
    }
  }, [connect])

  return { lastMessage, connected }
}
