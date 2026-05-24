import { NextRequest, NextResponse } from 'next/server'
import { createClient } from '@/lib/supabase/server'

const MANAGER_API_URL = process.env.MANAGER_API_URL || 'http://localhost:8090'
const MANAGER_TOKEN = process.env.MANAGER_TOKEN || ''
const MANAGER_WS_URL = process.env.MANAGER_WS_URL || ''

export async function GET(request: NextRequest) {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  // Request a short-lived ticket from the manager
  const ticketRes = await fetch(`${MANAGER_API_URL}/api/ws/ticket`, {
    headers: {
      'X-Manager-Token': MANAGER_TOKEN,
      'X-User-Id': user.id,
    },
  })
  if (!ticketRes.ok) {
    return NextResponse.json({ error: 'Failed to get WS ticket' }, { status: 502 })
  }
  const { ticket } = await ticketRes.json()
  if (!ticket) {
    return NextResponse.json({ error: 'No ticket' }, { status: 502 })
  }

  let wsBase: string
  if (MANAGER_WS_URL) {
    wsBase = MANAGER_WS_URL
  } else {
    const host = request.headers.get('x-forwarded-host') || request.headers.get('host') || 'localhost:8090'
    const protocol = request.headers.get('x-forwarded-proto') === 'https' ? 'wss' : 'ws'
    wsBase = `${protocol}://${host}`
  }

  const wsUrl = `${wsBase}/api/ws?ticket=${encodeURIComponent(ticket)}`
  return NextResponse.json({ wsUrl })
}
