'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { BacktestPanel } from '@/components/user/BacktestPanel'

export default function BacktestPage() {
  const t = useTranslations('Backtest')
  const [userId, setUserId] = useState('')

  useEffect(() => {
    const loadUser = async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data } = await supabase.auth.getUser()
      if (data.user) setUserId(data.user.id)
    }
    loadUser()
  }, [])

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>
      {userId && <BacktestPanel userId={userId} />}
    </div>
  )
}
