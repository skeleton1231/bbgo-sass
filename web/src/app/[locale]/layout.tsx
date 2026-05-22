import { NextIntlClientProvider, hasLocale } from 'next-intl'
import { getMessages, setRequestLocale } from 'next-intl/server'
import { notFound } from 'next/navigation'
import { routing } from '@/i18n/routing'

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }))
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  if (!hasLocale(routing.locales, locale)) {
    notFound()
  }

  setRequestLocale(locale)

  const messages = await getMessages()

  const clientNamespaces = [
    'Common', 'Errors', 'Auth', 'Nav', 'Dashboard', 'Bots', 'Backtest', 'Settings',
  ]
  const filteredMessages: Record<string, unknown> = {}
  for (const ns of clientNamespaces) {
    if (messages[ns]) filteredMessages[ns] = messages[ns]
  }

  return (
    <NextIntlClientProvider messages={filteredMessages}>
      {children}
    </NextIntlClientProvider>
  )
}
