'use client'

import { useTranslations } from 'next-intl'
import { Link, usePathname } from '@/i18n/navigation'
import { cn } from '@/lib/utils'

export default function SettingsPage() {
  const t = useTranslations('Settings')
  const pathname = usePathname()

  const tabs = [
    { href: './settings/api-keys', label: t('apiKeys.title') },
    { href: './settings/notifications', label: t('notifications.title') },
  ]

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <div className="flex gap-2">
        {tabs.map((tab) => (
          <Link
            key={tab.href}
            href={tab.href}
            className={cn(
              'rounded-md px-4 py-2 text-sm font-medium',
              pathname.endsWith(tab.href.split('/').pop()!) ? 'bg-primary text-primary-foreground' : 'border hover:bg-muted'
            )}
          >
            {tab.label}
          </Link>
        ))}
      </div>
    </div>
  )
}
