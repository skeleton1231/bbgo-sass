import { createServerClient } from '@supabase/ssr'
import { cookies } from 'next/headers'
import { cache } from 'react'
import type { Database } from './types'
import type { User } from '@supabase/supabase-js'

export async function createClient() {
  const cookieStore = await cookies()
  return createServerClient<Database>(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() {
          return cookieStore.getAll()
        },
        setAll(cookiesToSet) {
          try {
            cookiesToSet.forEach(({ name, value, options }) =>
              cookieStore.set(name, value, options)
            )
          } catch {
            // setAll called from Server Component where cookies cannot be set.
          }
        },
      },
    }
  )
}

export const getCurrentUser = cache(async (): Promise<User | null> => {
  const supabase = await createClient()
  const { data } = await supabase.auth.getUser()
  return data.user
})

type UserProfileRow = Database['public']['Tables']['user_profiles']['Row']

export const getUserRole = cache(async (): Promise<string | null> => {
  const user = await getCurrentUser()
  if (!user) return null

  const supabase = await createClient()
  const { data } = await supabase
    .from('user_profiles')
    .select('role')
    .eq('id', user.id)
    .single()

  return (data as Pick<UserProfileRow, 'role'> | null)?.role ?? null
})
