'use client'

import { createContext, useContext, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createClient as createSupabaseClient } from '@/lib/supabase/client'
import { registryRowToSchema, type StrategySchema } from '@/lib/bbgo/strategies'
import type { Database } from '@/lib/supabase/types'

type StrategyRegistryRow = Database['public']['Tables']['strategy_registry']['Row']

const StrategyRegistryContext = createContext<StrategySchema[]>([])

export function StrategyRegistryProvider({ children }: { children: ReactNode }) {
  const { data = [] } = useQuery<StrategySchema[]>({
    queryKey: ['strategy-registry'],
    queryFn: async () => {
      const sb = createSupabaseClient()
      const { data, error } = await sb
        .from('strategy_registry')
        .select('*')
        .eq('enabled', true)
        .order('sort_order')
      if (error) throw error
      return (data as StrategyRegistryRow[]).map(registryRowToSchema)
    },
    staleTime: 5 * 60_000,
  })

  return (
    <StrategyRegistryContext.Provider value={data}>
      {children}
    </StrategyRegistryContext.Provider>
  )
}

export function useStrategyRegistry(): StrategySchema[] {
  return useContext(StrategyRegistryContext)
}
