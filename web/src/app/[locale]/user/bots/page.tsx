'use client'

import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
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
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>
        </div>
        <Button onClick={() => setShowCreate(true)} className="rounded-full">
          <Plus className="mr-1.5 h-4 w-4" />
          {t('create')}
        </Button>
      </div>

      {userId && <StrategyList userId={userId} />}

      {showCreate && userId && (
        <CreateStrategyDialog userId={userId} onClose={() => setShowCreate(false)} />
      )}
    </div>
  )
}
