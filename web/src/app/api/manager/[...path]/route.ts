import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@/lib/supabase/server'

const MANAGER_URL = process.env.MANAGER_API_URL || 'http://localhost:8090'
const MANAGER_TOKEN = process.env.MANAGER_TOKEN || ''

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
    'X-Manager-Token': MANAGER_TOKEN,
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
  let userId: string
  try {
    const supabase = await createClient()
    const { data: { user } } = await supabase.auth.getUser()
    if (!user) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }
    userId = user.id
  } catch {
    return NextResponse.json({ error: 'Auth service unavailable' }, { status: 503 })
  }

  const { path } = await params
  const res = await forwardRequest(method, request, path, userId)
  const text = await res.text()

  try {
    const data = JSON.parse(text)
    return NextResponse.json(data, { status: res.status })
  } catch {
    return NextResponse.json(
      { error: `Manager returned non-JSON: ${text.slice(0, 200)}` },
      { status: res.status }
    )
  }
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

export async function PUT(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return handleRequest('PUT', request, params)
}

export async function PATCH(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return handleRequest('PATCH', request, params)
}
