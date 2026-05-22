'use client'

import { useTranslations } from 'next-intl'
import Link from 'next/link'
import { usePathname } from '@/i18n/navigation'
import { cn } from '@/lib/utils'
import { signOutAction } from '@/actions/auth'

const navItems = [
  { key: 'dashboard', href: '/user' },
  { key: 'bots', href: '/user/bots' },
  { key: 'backtest', href: '/user/backtest' },
  { key: 'settings', href: '/user/settings' },
] as const

export function Sidebar() {
  const t = useTranslations('Nav')
  const pathname = usePathname()

  return (
    <aside className="flex h-screen w-60 flex-col border-r bg-card">
      <div className="flex h-14 items-center border-b px-4">
        <span className="text-lg font-bold">BBGO</span>
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {navItems.map(({ key, href }) => (
          <Link
            key={key}
            href={href}
            className={cn(
              'block rounded-md px-3 py-2 text-sm transition-colors',
              pathname === href
                ? 'bg-primary/10 text-primary font-medium'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            )}
          >
            {t(key)}
          </Link>
        ))}
      </nav>

      <div className="border-t p-2">
        <form action={signOutAction}>
          <button
            type="submit"
            className="w-full rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground transition-colors text-left"
          >
            {t('signOut')}
          </button>
        </form>
      </div>
    </aside>
  )
}
