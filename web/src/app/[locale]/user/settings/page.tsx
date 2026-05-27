'use client'

import { useTranslations } from 'next-intl'
import { Link, usePathname } from '@/i18n/navigation'
import { cn } from '@/lib/utils'
import { KeyRound, Bell } from 'lucide-react'

const tabs = [
  { href: '/user/settings/api-keys', key: 'apiKeys', icon: KeyRound },
  { href: '/user/settings/notifications', key: 'notifications', icon: Bell },
] as const

export default function SettingsPage() {
  const t = useTranslations('Settings')
  const pathname = usePathname()

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">{t('title')}</h1>

      <div className="flex gap-2">
        {tabs.map(({ href, key, icon: Icon }) => {
          const active = pathname.endsWith(href.split('/').pop()!)
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                'inline-flex items-center gap-2 rounded-full px-4 py-2 text-sm font-medium transition-colors',
                active
                  ? 'bg-primary text-primary-foreground'
                  : 'border hover:bg-muted'
              )}
            >
              <Icon className="h-4 w-4" />
              {t(`${key}.title`)}
            </Link>
          )
        })}
      </div>
    </div>
  )
}
