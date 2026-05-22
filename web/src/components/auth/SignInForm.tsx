'use client'

import { useActionState } from 'react'
import { useTranslations } from 'next-intl'
import { signInAction } from '@/actions/auth'

export function SignInForm() {
  const t = useTranslations('Auth')
  const [state, formAction, isPending] = useActionState(signInAction, null)

  return (
    <form action={formAction} className="space-y-4">
      <div>
        <label htmlFor="email" className="block text-sm font-medium mb-1">
          {t('email')}
        </label>
        <input
          id="email"
          name="email"
          type="email"
          required
          placeholder={t('emailPlaceholder')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        />
      </div>

      <div>
        <label htmlFor="password" className="block text-sm font-medium mb-1">
          {t('password')}
        </label>
        <input
          id="password"
          name="password"
          type="password"
          required
          minLength={6}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        />
      </div>

      {state?.error && (
        <p className="text-sm text-destructive">{state.error}</p>
      )}

      <button
        type="submit"
        disabled={isPending}
        className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {isPending ? t('loggingIn') : t('login')}
      </button>
    </form>
  )
}
