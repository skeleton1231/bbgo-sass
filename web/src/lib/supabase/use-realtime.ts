'use client'

import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { createClient } from '@/lib/supabase/client'

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/**
 * Subscribes to Supabase Realtime INSERT events on a table filtered by user_id.
 * On each event, invalidates the specified TanStack Query keys so they refetch.
 */
export function useRealtimeTable(
  table: string,
  userId: string | undefined,
  queryKeysToInvalidate: string[][],
  opts?: { mode?: 'live' | 'paper'; enabled?: boolean }
) {
  const queryClient = useQueryClient()
  const channelRef = useRef<ReturnType<ReturnType<typeof createClient>['channel']> | null>(null)

  const enabled = opts?.enabled !== false && !!userId && UUID_RE.test(userId ?? '')
  const tableName = opts?.mode === 'paper' ? `paper_${table}` : table
  const keysJson = JSON.stringify(queryKeysToInvalidate)

  useEffect(() => {
    if (!enabled) return

    const sb = createClient()
    const channelName = `realtime-${tableName}-${userId}`
    const keys = JSON.parse(keysJson) as string[][]

    const channel = sb
      .channel(channelName)
      .on(
        'postgres_changes',
        {
          event: 'INSERT',
          schema: 'public',
          table: tableName,
          filter: `user_id=eq.${userId}`,
        },
        () => {
          for (const key of keys) {
            queryClient.invalidateQueries({ queryKey: key })
          }
        }
      )
      .subscribe()

    channelRef.current = channel

    return () => {
      sb.removeChannel(channel)
      channelRef.current = null
    }
  }, [enabled, tableName, userId, queryClient, keysJson])
}
