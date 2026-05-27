'use client'

import { useTranslations } from 'next-intl'
import { ApiKeyList } from '@/components/user/ApiKeyList'

export default function ApiKeysPage() {
  const t = useTranslations('Settings.apiKeys')
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>
      <ApiKeyList />
    </div>
  )
}
