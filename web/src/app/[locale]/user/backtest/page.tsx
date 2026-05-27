'use client'

import { useTranslations } from 'next-intl'
import { BacktestPanel } from '@/components/user/BacktestPanel'
import { useUserId } from '@/components/providers/user-id'

export default function BacktestPage() {
  const t = useTranslations('Backtest')
  const userId = useUserId()

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>
      <BacktestPanel userId={userId} />
    </div>
  )
}
