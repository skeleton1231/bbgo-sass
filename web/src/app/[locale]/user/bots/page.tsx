'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { StrategyList } from '@/components/user/StrategyList'
import { CreateStrategyDialog } from '@/components/user/CreateStrategyDialog'

export default function BotsPage() {
  const t = useTranslations('Bots')
  const [showCreate, setShowCreate] = useState(false)
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
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">{t('title')}</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          {t('create')}
        </button>
      </div>

      {userId && <StrategyList userId={userId} />}

      {showCreate && userId && (
        <CreateStrategyDialog userId={userId} onClose={() => setShowCreate(false)} />
      )}
    </div>
  )
}
