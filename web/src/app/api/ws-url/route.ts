import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@/lib/supabase/server'

const MANAGER_TOKEN = process.env.MANAGER_TOKEN || ''
const MANAGER_WS_URL = process.env.MANAGER_WS_URL || ''

export async function GET(request: NextRequest) {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  let wsUrl: string

  if (MANAGER_WS_URL) {
    wsUrl = `${MANAGER_WS_URL}/api/ws?token=${encodeURIComponent(MANAGER_TOKEN)}&userId=${encodeURIComponent(user.id)}`
  } else {
    const host = request.headers.get('x-forwarded-host') || request.headers.get('host') || 'localhost:8090'
    const protocol = request.headers.get('x-forwarded-proto') === 'https' ? 'wss' : 'ws'
    wsUrl = `${protocol}://${host}/api/ws?token=${encodeURIComponent(MANAGER_TOKEN)}&userId=${encodeURIComponent(user.id)}`
  }

  return NextResponse.json({ wsUrl })
}
