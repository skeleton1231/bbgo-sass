'use client'

import { useTranslations } from 'next-intl'
import { ApiKeyList } from '@/components/user/ApiKeyList'

export default function ApiKeysPage() {
  const t = useTranslations('Settings.apiKeys')
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>
      <ApiKeyList />
    </div>
  )
}
