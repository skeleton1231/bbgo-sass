'use client'

import { createContext, useContext, useCallback, useEffect, useSyncExternalStore } from 'react'
import { useSearchParams } from 'next/navigation'

export type TradingMode = 'live' | 'paper'

const STORAGE_KEY = 'bbgo-trading-mode'
const DEFAULT_MODE: TradingMode = 'live'

function readStoredMode(): TradingMode {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'live' || stored === 'paper') return stored
  return DEFAULT_MODE
}

function getServerSnapshot(): TradingMode {
  return DEFAULT_MODE
}

function subscribe(callback: () => void) {
  window.addEventListener('storage', callback)
  return () => window.removeEventListener('storage', callback)
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

  const storedMode = useSyncExternalStore(subscribe, readStoredMode, getServerSnapshot)

  // Persist URL ?mode= to localStorage — useSyncExternalStore picks up the change
  useEffect(() => {
    if (urlMode === 'live' || urlMode === 'paper') {
      const current = localStorage.getItem(STORAGE_KEY)
      if (current !== urlMode) {
        localStorage.setItem(STORAGE_KEY, urlMode)
        window.dispatchEvent(new StorageEvent('storage', { key: STORAGE_KEY, newValue: urlMode }))
      }
    }
  }, [urlMode])

  const setMode = useCallback((m: TradingMode) => {
    localStorage.setItem(STORAGE_KEY, m)
    window.dispatchEvent(new StorageEvent('storage', { key: STORAGE_KEY, newValue: m }))
  }, [])

  return (
    <TradingModeContext.Provider value={{ mode: storedMode, setMode }}>
      {children}
    </TradingModeContext.Provider>
  )
}

export function useTradingMode() {
  return useContext(TradingModeContext)
}
