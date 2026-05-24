import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@/lib/supabase/server'

const MANAGER_TOKEN = process.env.MANAGER_TOKEN || ''
const MANAGER_WS_HOST = process.env.MANAGER_WS_HOST || 'localhost:8090'

function extractPort(host: string): string {
  const parts = host.split(':')
  return parts.length > 1 ? parts[parts.length - 1]! : '8090'
}

export async function GET(request: NextRequest) {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  const host = request.headers.get('x-forwarded-host') || request.headers.get('host') || MANAGER_WS_HOST
  const protocol = request.headers.get('x-forwarded-proto') === 'https' ? 'wss' : 'ws'
  const port = extractPort(MANAGER_WS_HOST)
  const wsUrl = `${protocol}://${host.replace(/:\d+$/, '')}:${port}/api/ws?token=${encodeURIComponent(MANAGER_TOKEN)}&userId=${encodeURIComponent(user.id)}`

  return NextResponse.json({ wsUrl })
}
