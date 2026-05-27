import { getCurrentUser } from '@/lib/supabase/server'
import { redirect } from '@/i18n/navigation'
import { getLocale } from 'next-intl/server'
import { LOGIN_PATH } from '@/lib/routes'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'

export default async function UserLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const [user, locale] = await Promise.all([
    getCurrentUser(),
    getLocale(),
  ])

  if (!user) {
    redirect({ href: LOGIN_PATH, locale })
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-auto p-6 lg:p-8">{children}</main>
      </div>
    </div>
  )
}
