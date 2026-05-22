import { NextRequest, NextResponse } from 'next/server'
import { createServerClient } from '@supabase/ssr'

const MANAGER_URL = process.env.MANAGER_API_URL || 'http://localhost:8090'

async function verifyAuth(request: NextRequest): Promise<string | null> {
  const cookieStore = request.cookies
  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() { return cookieStore.getAll() },
        setAll() {},
      },
    }
  )
  const { data } = await supabase.auth.getUser()
  return data.user?.id ?? null
}

function forwardRequest(
  method: string,
  request: NextRequest,
  path: string[],
  userId: string
) {
  const search = request.nextUrl.search
  const targetUrl = `${MANAGER_URL}/api/${path.join('/')}${search}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-User-Id': userId,
  }

  const init: RequestInit = { method, headers }

  if (method !== 'GET' && method !== 'HEAD') {
    return request.text().then((body) => {
      init.body = body
      return fetch(targetUrl, init)
    })
  }

  return fetch(targetUrl, init)
}

async function handleRequest(
  method: string,
  request: NextRequest,
  params: Promise<{ path: string[] }>
) {
  const userId = await verifyAuth(request)
  if (!userId) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  const { path } = await params
  const res = await forwardRequest(method, request, path, userId)
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}

export async function GET(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return handleRequest('GET', request, params)
}

export async function POST(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return handleRequest('POST', request, params)
}

export async function DELETE(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return handleRequest('DELETE', request, params)
}
