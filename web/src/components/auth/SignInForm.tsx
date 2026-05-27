'use client'

import { useActionState } from 'react'
import { useTranslations } from 'next-intl'
import { signInAction } from '@/actions/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Link } from '@/i18n/navigation'

export function SignInForm() {
  const t = useTranslations('Auth')
  const [state, formAction, isPending] = useActionState(signInAction, null)

  return (
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
        <div className="flex items-center justify-between">
          <Label htmlFor="password">{t('password')}</Label>
          <Link
            href="/forgot-password"
            className="text-xs text-primary hover:underline"
          >
            {t('forgotPassword')}
          </Link>
        </div>
        <Input
          id="password"
          name="password"
          type="password"
          autoComplete="current-password"
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
        {isPending ? t('loggingIn') : t('login')}
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        {t('noAccount')}{' '}
        <Link href="/signup" className="text-primary hover:underline font-medium">
          {t('signUp')}
        </Link>
      </p>
    </form>
  )
}
