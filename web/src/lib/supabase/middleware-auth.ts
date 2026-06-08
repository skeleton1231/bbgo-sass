import { createServerClient } from '@supabase/ssr'
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import type { Database } from './types'
import { routing } from '@/i18n/routing'
import { LOGIN_PATH, SIGNUP_PATH, USER_PATH } from '@/lib/routes'

const SUPABASE_URL = process.env.NEXT_PUBLIC_SUPABASE_URL
const SUPABASE_ANON_KEY = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY
if (!SUPABASE_URL) throw new Error('NEXT_PUBLIC_SUPABASE_URL is not set')
if (!SUPABASE_ANON_KEY) throw new Error('NEXT_PUBLIC_SUPABASE_ANON_KEY is not set')
const validatedUrl: string = SUPABASE_URL
const validatedKey: string = SUPABASE_ANON_KEY

const COOKIE_MAX_AGE = 60 * 60 * 24 * 7

const secureCookieOptions = {
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'lax' as const,
  maxAge: COOKIE_MAX_AGE,
  path: '/',
}

function stripLocale(path: string) {
  for (const locale of routing.locales) {
    if (path === `/${locale}`) return '/'
    if (path.startsWith(`/${locale}/`)) return path.slice(locale.length + 1)
  }
  return path
}

function detectLocale(pathname: string): string | null {
  for (const locale of routing.locales) {
    if (pathname === `/${locale}` || pathname.startsWith(`/${locale}/`)) return locale
  }
  return null
}

function localePrefixedPath(barePath: string, locale: string | null): string {
  if (!locale || locale === routing.defaultLocale) return barePath
  return `/${locale}${barePath}`
}

function redirectWithCookies(request: NextRequest, reply: NextResponse, targetPath: string, locale: string | null) {
  const url = request.nextUrl.clone()
  url.pathname = localePrefixedPath(targetPath, locale)
  const redirectResponse = NextResponse.redirect(url)
  reply.cookies.getAll().forEach((cookie) => {
    redirectResponse.cookies.set(cookie.name, cookie.value, secureCookieOptions)
  })
  return redirectResponse
}

export async function updateSession(request: NextRequest, response: NextResponse) {
  const reply = response

  const supabase = createServerClient<Database>(
    validatedUrl,
    validatedKey,
    {
      cookies: {
        getAll() {
          return request.cookies.getAll()
        },
        setAll(cookiesToSet) {
          cookiesToSet.forEach(({ name, value }) =>
            request.cookies.set(name, value)
          )
          cookiesToSet.forEach(({ name, value, options }) => {
            reply.cookies.set(name, value, { ...options, ...secureCookieOptions })
          })
        },
      },
    }
  )

  const pathname = request.nextUrl.pathname
  const barePath = stripLocale(pathname)
  const locale = detectLocale(pathname)

  const isProtectedRoute = barePath.startsWith(USER_PATH)
  const isAuthRoute = barePath === LOGIN_PATH || barePath === SIGNUP_PATH

  let user: Awaited<ReturnType<typeof supabase.auth.getUser>>['data']['user'] | null = null
  try {
    const { data } = await supabase.auth.getUser()
    user = data?.user ?? null
    // Force session refresh to rewrite cookies without httpOnly flag
    if (user) {
      await supabase.auth.refreshSession()
    }
  } catch {
    // Network/auth service error — don't kick user out, just continue
    return reply
  }

  if (isProtectedRoute && !user) {
    return redirectWithCookies(request, reply, LOGIN_PATH, locale)
  }

  if (user && isAuthRoute) {
    return redirectWithCookies(request, reply, USER_PATH, locale)
  }

  return reply
}
