'use client'

import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { useSearchParams } from 'next/navigation'

export type TradingMode = 'live' | 'paper'

const STORAGE_KEY = 'bbgo-trading-mode'
const DEFAULT_MODE: TradingMode = 'live'

function readStoredMode(): TradingMode {
  if (typeof window === 'undefined') return DEFAULT_MODE
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'live' || stored === 'paper') return stored
  return DEFAULT_MODE
}

const TradingModeContext = createContext<{
  mode: TradingMode
  setMode: (mode: TradingMode) => void
}>({
  mode: DEFAULT_MODE,
  setMode: () => {},
})

export function TradingModeProvider({ children }: { children: React.ReactNode }) {
  const searchParams = useSearchParams()
  const urlMode = searchParams.get('mode')
  const [mode, setModeState] = useState<TradingMode>(() => {
    if (typeof window === 'undefined') return DEFAULT_MODE
    if (urlMode === 'live' || urlMode === 'paper') {
      localStorage.setItem(STORAGE_KEY, urlMode)
      return urlMode
    }
    return readStoredMode()
  })

  // React to URL ?mode= changes on SPA navigation (not just initial mount)
  if ((urlMode === 'live' || urlMode === 'paper') && urlMode !== mode) {
    localStorage.setItem(STORAGE_KEY, urlMode)
    setModeState(urlMode)
  }

  const setMode = useCallback((m: TradingMode) => {
    setModeState(m)
    localStorage.setItem(STORAGE_KEY, m)
  }, [])

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
