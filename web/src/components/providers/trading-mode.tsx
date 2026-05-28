'use client'

import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { useSearchParams } from 'next/navigation'

export type TradingMode = 'live' | 'paper'

const STORAGE_KEY = 'bbgo-trading-mode'
const DEFAULT_MODE: TradingMode = 'live'

const TradingModeContext = createContext<{
  mode: TradingMode
  setMode: (mode: TradingMode) => void
}>({
  mode: DEFAULT_MODE,
  setMode: () => {},
})

export function TradingModeProvider({ children }: { children: React.ReactNode }) {
  const searchParams = useSearchParams()
  const [mode, setModeState] = useState<TradingMode>(DEFAULT_MODE)

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'live' || stored === 'paper') setModeState(stored)
  }, [])

  const setMode = useCallback((m: TradingMode) => {
    setModeState(m)
    localStorage.setItem(STORAGE_KEY, m)
  }, [])

  useEffect(() => {
    const urlMode = searchParams.get('mode')
    if (urlMode === 'live' || urlMode === 'paper') {
      setModeState(urlMode)
      localStorage.setItem(STORAGE_KEY, urlMode)
    }
  }, [searchParams])

  useEffect(() => {
    const handler = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY && (e.newValue === 'live' || e.newValue === 'paper')) {
        setModeState(e.newValue)
      }
    }
    window.addEventListener('storage', handler)
    return () => window.removeEventListener('storage', handler)
  }, [])

  return (
    <TradingModeContext.Provider value={{ mode, setMode }}>
      {children}
    </TradingModeContext.Provider>
  )
}

export function useTradingMode() {
  return useContext(TradingModeContext)
}
