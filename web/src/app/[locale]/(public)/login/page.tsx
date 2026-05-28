import { getTranslations } from 'next-intl/server'
import { SignInForm } from '@/components/auth/SignInForm'
import { SessionExpiredBanner } from '@/components/auth/SessionExpiredBanner'
import { Activity } from 'lucide-react'

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ signup?: string }>
}) {
  const t = await getTranslations('Auth')
  const params = await searchParams
  const showSignupSuccess = params.signup === '1'

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
            <h2 className="text-xl font-semibold">{t('welcomeBack')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('loginSubtitle')}
            </p>
          </div>
          {showSignupSuccess && (
            <div className="rounded-lg bg-primary/10 px-3 py-2">
              <p className="text-sm text-primary">{t('signupSuccess')}</p>
            </div>
          )}
          <SessionExpiredBanner />
          <SignInForm />
        </div>
      </div>
    </div>
  )
}
