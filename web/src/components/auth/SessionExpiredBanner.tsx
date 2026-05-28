'use client'

import { useEffect, useState } from 'react'
import { useTranslations } from 'next-intl'
import { AlertCircle } from 'lucide-react'

export function SessionExpiredBanner() {
  const t = useTranslations('Auth')
  const [show, setShow] = useState(false)

  useEffect(() => {
    const msg = sessionStorage.getItem('bbgo-auth-message')
    if (msg === 'session_expired') {
      setShow(true)
      sessionStorage.removeItem('bbgo-auth-message')
    }
  }, [])

  if (!show) return null

  return (
    <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2">
      <AlertCircle className="h-4 w-4 shrink-0 text-destructive" />
      <p className="text-sm text-destructive">{t('sessionExpired')}</p>
      <button
        type="button"
        onClick={() => setShow(false)}
        className="ml-auto text-destructive/60 hover:text-destructive"
        aria-label="Dismiss"
      >
        &times;
      </button>
    </div>
  )
}
