'use client'

import { useTranslations } from 'next-intl'
import { Link, usePathname, useRouter } from '@/i18n/navigation'
import { cn } from '@/lib/utils'
import { createClient } from '@/lib/supabase/client'
import { LOGIN_PATH } from '@/lib/routes'
import { Separator } from '@/components/ui/separator'
import {
  LayoutDashboard,
  Bot,
  FlaskConical,
  Settings,
  LogOut,
  Activity,
} from 'lucide-react'

const navItems = [
  { key: 'dashboard', href: '/user', icon: LayoutDashboard },
  { key: 'bots', href: '/user/bots', icon: Bot },
  { key: 'backtest', href: '/user/backtest', icon: FlaskConical },
  { key: 'settings', href: '/user/settings', icon: Settings },
] as const

function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const t = useTranslations('Nav')
  const pathname = usePathname()
  const router = useRouter()

  async function handleSignOut() {
    sessionStorage.setItem('bbgo-auth-message', 'signed_out')
    const supabase = createClient()
    await supabase.auth.signOut()
    router.push(LOGIN_PATH)
    router.refresh()
  }

  return (
    <>
      <nav className="flex-1 space-y-1 p-3">
        {navItems.map(({ key, href, icon: Icon }) => {
          const isActive =
            pathname === href || (key !== 'dashboard' && pathname.startsWith(href))
          return (
            <Link
              key={key}
              href={href}
              onClick={onNavigate}
              className={cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-[13px] font-medium transition-colors',
                isActive
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                  : 'text-sidebar-foreground/60 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground'
              )}
            >
              <Icon className="h-4 w-4" />
              {t(key)}
            </Link>
          )
        })}
      </nav>

      <Separator className="bg-sidebar-border" />

      <div className="p-3">
        <button
          type="button"
          onClick={handleSignOut}
          className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-[13px] font-medium text-sidebar-foreground/60 transition-colors hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"
        >
          <LogOut className="h-4 w-4" />
          {t('signOut')}
        </button>
      </div>
    </>
  )
}

export function Sidebar() {
  return (
    <aside className="hidden md:flex h-screen w-[240px] flex-col bg-sidebar text-sidebar-foreground border-r border-sidebar-border">
      <div className="flex h-16 items-center gap-2.5 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
          <Activity className="h-4 w-4 text-primary-foreground" />
        </div>
        <span className="text-[15px] font-semibold tracking-tight">BBGO</span>
      </div>

      <Separator className="bg-sidebar-border" />

      <SidebarNav />
    </aside>
  )
}

export function MobileSidebar() {
  return (
    <div className="flex h-full flex-col bg-sidebar text-sidebar-foreground">
      <div className="flex h-16 items-center gap-2.5 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
          <Activity className="h-4 w-4 text-primary-foreground" />
        </div>
        <span className="text-[15px] font-semibold tracking-tight">BBGO</span>
      </div>

      <Separator className="bg-sidebar-border" />

      <SidebarNav />
    </div>
  )
}
