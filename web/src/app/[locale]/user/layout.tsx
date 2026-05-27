import { getCurrentUser } from '@/lib/supabase/server'
import { redirect } from 'next/navigation'
import { getLocale } from 'next-intl/server'
import { LOGIN_PATH } from '@/lib/routes'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'
import { UserIdProvider } from '@/components/providers/user-id'

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
    const prefix = locale === 'en' ? '' : `/${locale}`
    redirect(`${prefix}${LOGIN_PATH}`)
  }

  return (
    <UserIdProvider userId={user.id}>
      <div className="flex h-screen">
        <Sidebar />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header email={user.email} />
          <main className="flex-1 overflow-auto p-6 lg:p-8">{children}</main>
        </div>
      </div>
    </UserIdProvider>
  )
}
