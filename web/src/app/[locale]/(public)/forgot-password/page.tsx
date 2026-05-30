'use client'

import { useActionState } from 'react'
import { useTranslations } from 'next-intl'
import { resetPasswordAction } from '@/actions/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Link } from '@/i18n/navigation'
import { Activity, AlertCircle } from 'lucide-react'

export default function ForgotPasswordPage() {
  const t = useTranslations('Auth')
  const [state, formAction, isPending] = useActionState(resetPasswordAction, null)

  return (
    <div className="flex min-h-screen items-center justify-center p-8">
      <div className="w-full max-w-sm space-y-8">
        <div className="text-center">
          <div className="flex items-center justify-center gap-2 mb-6">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary">
              <Activity className="h-4 w-4 text-primary-foreground" />
            </div>
            <span className="text-lg font-semibold tracking-tight">BBGO</span>
          </div>
          <h2 className="text-xl font-semibold">{t('resetPassword')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('resetPasswordSubtitle')}
          </p>
        </div>

        <form action={formAction} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">{t('email')}</Label>
            <Input
              id="email"
              name="email"
              type="email"
              required
              placeholder={t('emailPlaceholder')}
              className="h-11 rounded-lg"
            />
          </div>

          {state?.error && (
            <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2">
              <AlertCircle className="h-4 w-4 shrink-0 text-destructive" />
              <p className="text-sm text-destructive">{state.error}</p>
            </div>
          )}

          {state?.success && (
            <div className="rounded-lg bg-primary/10 px-3 py-2">
              <p className="text-sm text-primary">{state.success}</p>
            </div>
          )}

          <Button
            type="submit"
            disabled={isPending}
            className="w-full rounded-full h-11"
          >
            {isPending ? t('sending') : t('sendResetLink')}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          <Link href="/login" className="text-primary hover:underline font-medium">
            {t('backToLogin')}
          </Link>
        </p>
      </div>
    </div>
  )
}
