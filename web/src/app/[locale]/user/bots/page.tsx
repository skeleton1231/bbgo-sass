'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import { StrategyList } from '@/components/user/StrategyList'
import { CreateStrategyDialog } from '@/components/user/CreateStrategyDialog'
import { useUserId } from '@/components/providers/user-id'

export default function BotsPage() {
  const t = useTranslations('Bots')
  const userId = useUserId()
  const [showCreate, setShowCreate] = useState(false)

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
