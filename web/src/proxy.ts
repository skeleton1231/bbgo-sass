import { type NextRequest, NextResponse } from 'next/server'
import createIntlMiddleware from 'next-intl/middleware'
import { routing } from '@/i18n/routing'
import { updateSession } from '@/lib/supabase/middleware-auth'

const handleI18nRouting = createIntlMiddleware(routing)

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Bypass for API routes
  if (pathname === '/api' || pathname.startsWith('/api/')) {
    return NextResponse.next()
  }

  // Bypass for auth callback (handles its own session via route handler)
  if (pathname.startsWith('/auth/')) {
    return NextResponse.next()
  }

  // Static assets pass through
  if (
    pathname.startsWith('/_next/') ||
    pathname === '/favicon.ico' ||
    pathname === '/sitemap.xml' ||
    pathname === '/robots.txt'
  ) {
    return NextResponse.next()
  }

  // Apply i18n routing first
  const response = handleI18nRouting(request)

  // Delegate auth/session handling to supabase middleware
  return updateSession(request, response)
}
