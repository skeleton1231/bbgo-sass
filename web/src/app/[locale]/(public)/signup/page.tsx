'use client'

import { useActionState } from 'react'
import { useTranslations } from 'next-intl'
import { signUpAction } from '@/actions/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Link } from '@/i18n/navigation'
import { Activity } from 'lucide-react'

export default function SignupPage() {
  const t = useTranslations('Auth')
  const [state, formAction, isPending] = useActionState(signUpAction, null)

  return (
    <div className="flex min-h-screen">
      <div className="hidden lg:flex lg:w-1/2 bg-surface-dark items-center justify-center p-12">
        <div className="max-w-md space-y-8">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary">
              <Activity className="h-5 w-5 text-primary-foreground" />
            </div>
            <span className="text-xl font-semibold text-white tracking-tight">BBGO</span>
          </div>
          <h1 className="text-4xl font-normal leading-tight text-white" style={{ letterSpacing: '-1px' }}>
            {t('heroTitle')}
          </h1>
          <p className="text-[15px] leading-relaxed text-white/50">
            {t('heroSubtitle')}
          </p>
        </div>
      </div>

      <div className="flex flex-1 items-center justify-center p-8">
        <div className="w-full max-w-sm space-y-8">
          <div className="text-center">
            <div className="flex items-center justify-center gap-2 lg:hidden mb-6">
              <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary">
                <Activity className="h-4.5 w-4.5 text-primary-foreground" />
              </div>
              <span className="text-lg font-semibold tracking-tight">BBGO</span>
            </div>
            <h2 className="text-xl font-semibold">{t('createAccount')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('signupSubtitle')}
            </p>
          </div>

          <form action={formAction} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">{t('email')}</Label>
              <Input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                placeholder={t('emailPlaceholder')}
                className="h-11 rounded-lg"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">{t('password')}</Label>
              <Input
                id="password"
                name="password"
                type="password"
                autoComplete="new-password"
                required
                minLength={6}
                className="h-11 rounded-lg"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">{t('confirmPassword')}</Label>
              <Input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                autoComplete="new-password"
                required
                minLength={6}
                className="h-11 rounded-lg"
              />
            </div>

            {state?.error && (
              <div className="rounded-lg bg-destructive/10 px-3 py-2">
                <p className="text-sm text-destructive">{state.error}</p>
              </div>
            )}

            <Button
              type="submit"
              disabled={isPending}
              className="w-full rounded-full h-11"
            >
              {isPending ? t('signingUp') : t('signUp')}
            </Button>
          </form>

          <p className="text-center text-sm text-muted-foreground">
            {t('hasAccount')}{' '}
            <Link href="/login" className="text-primary hover:underline font-medium">
              {t('login')}
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
