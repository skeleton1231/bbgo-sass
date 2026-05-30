'use client'

import { useTranslations } from 'next-intl'
import { BacktestPanel } from '@/components/user/BacktestPanel'

export default function BacktestPage() {
  const t = useTranslations('Backtest')

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>
      <BacktestPanel />
    </div>
  )
}
