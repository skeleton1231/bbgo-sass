import { type NextRequest, NextResponse } from 'next/server'
import createIntlMiddleware from 'next-intl/middleware'
import { routing } from '@/i18n/routing'
import { updateSession } from '@/lib/supabase/middleware-auth'

const handleI18nRouting = createIntlMiddleware(routing)

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (pathname === '/api' || pathname.startsWith('/api/')) {
    return NextResponse.next()
  }

  if (pathname.startsWith('/auth/')) {
    return NextResponse.next()
  }

  if (
    pathname.startsWith('/_next/') ||
    pathname === '/favicon.ico' ||
    pathname === '/sitemap.xml' ||
    pathname === '/robots.txt'
  ) {
    return NextResponse.next()
  }

  const response = handleI18nRouting(request)

  return updateSession(request, response)
}
