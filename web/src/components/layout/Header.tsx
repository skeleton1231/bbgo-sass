'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { usePathname } from '@/i18n/navigation'
import { UserNav } from './UserNav'
import { MobileSidebar } from './Sidebar'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetTitle,
} from '@/components/ui/sheet'
import { Menu } from 'lucide-react'

const breadcrumbMap: Record<string, string> = {
  '/user': 'dashboard',
  '/user/bots': 'bots',
  '/user/backtest': 'backtest',
  '/user/settings': 'settings',
  '/user/settings/api-keys': 'apiKeys',
  '/user/settings/notifications': 'notifications',
}

export function Header() {
  const t = useTranslations('Nav')
  const pathname = usePathname()
  const [mobileOpen, setMobileOpen] = useState(false)

  const segments = pathname.split('/').filter(Boolean)
  const crumbs = segments
    .map((_, i) => '/' + segments.slice(0, i + 1).join('/'))
    .filter((p) => breadcrumbMap[p])

  return (
    <>
      <header className="flex h-14 items-center justify-between border-b bg-background px-4 md:px-6">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setMobileOpen(true)}
          >
            <Menu className="h-5 w-5" />
            <span className="sr-only">{t('openMenu')}</span>
          </Button>

          <nav className="flex items-center gap-1.5 text-[13px]">
            {crumbs.map((crumb, i) => (
              <span key={crumb} className="flex items-center gap-1.5">
                {i > 0 && <span className="text-muted-foreground/40">/</span>}
                <span
                  className={
                    i === crumbs.length - 1
                      ? 'font-medium text-foreground'
                      : 'text-muted-foreground'
                  }
                >
                  {t(breadcrumbMap[crumb] ?? crumb)}
                </span>
              </span>
            ))}
          </nav>
        </div>
        <UserNav />
      </header>

      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="w-[240px] p-0" showCloseButton={false}>
          <SheetTitle className="sr-only">{t('navigation')}</SheetTitle>
          <MobileSidebar />
        </SheetContent>
      </Sheet>
    </>
  )
}
