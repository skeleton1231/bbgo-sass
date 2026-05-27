'use client'

import { useTranslations } from 'next-intl'
import { Link } from '@/i18n/navigation'
import { Button } from '@/components/ui/button'
import { Activity } from 'lucide-react'

export default function NotFound() {
  const t = useTranslations('Common')

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background p-8">
      <div className="flex items-center gap-2 mb-8">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary">
          <Activity className="h-4 w-4 text-primary-foreground" />
        </div>
        <span className="text-lg font-semibold tracking-tight">BBGO</span>
      </div>
      <h1 className="text-6xl font-semibold text-muted-foreground/30">404</h1>
      <p className="mt-3 text-lg text-muted-foreground">{t('pageNotFound')}</p>
      <Link href="/user" className="mt-6">
        <Button className="rounded-full px-6">{t('goHome')}</Button>
      </Link>
    </div>
  )
}
