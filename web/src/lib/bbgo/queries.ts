import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchUserStrategies,
  createStrategy as apiCreateStrategy,
  deleteStrategy as apiDeleteStrategy,
  startUser as apiStartUser,
  stopUser as apiStopUser,
  fetchUserStatus,
  runBacktest as apiRunBacktest,
  fetchCredentials as apiFetchCredentials,
  createCredential as apiCreateCredential,
  deleteCredential as apiDeleteCredential,
  fetchBotSessions,
  fetchBotOpenOrders,
  fetchBotTrades,
  fetchBotAccount,
  type StrategyEntry,
  type UserContainer,
  type BacktestResult,
  type CredentialInfo,
} from './manager'

export function useUserStrategies(userId: string) {
  return useQuery<UserContainer>({
    queryKey: ['user-strategies', userId],
    queryFn: () => fetchUserStrategies(userId),
    enabled: !!userId,
    refetchInterval: 10_000,
  })
}

export function useCreateStrategy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, ...data }: { userId: string } & Parameters<typeof apiCreateStrategy>[1]) =>
      apiCreateStrategy(userId, data),
    onSuccess: (_data, variables) => qc.invalidateQueries({ queryKey: ['user-strategies', variables.userId] }),
  })
}

export function useDeleteStrategy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, strategyId }: { userId: string; strategyId: string }) =>
      apiDeleteStrategy(userId, strategyId),
    onSuccess: (_data, variables) => qc.invalidateQueries({ queryKey: ['user-strategies', variables.userId] }),
  })
}

export function useStartUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: apiStartUser,
    onSuccess: (_data, userId) => qc.invalidateQueries({ queryKey: ['user-strategies', userId] }),
  })
}

export function useStopUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: apiStopUser,
    onSuccess: (_data, userId) => qc.invalidateQueries({ queryKey: ['user-strategies', userId] }),
  })
}

export function useRunBacktest() {
  return useMutation<BacktestResult, Error, Parameters<typeof apiRunBacktest>[0]>({
    mutationFn: apiRunBacktest,
  })
}

export function useBotSessions(userId: string) {
  return useQuery({
    queryKey: ['bot-sessions', userId],
    queryFn: () => fetchBotSessions(userId),
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useBotOpenOrders(userId: string, session: string) {
  return useQuery({
    queryKey: ['bot-orders', userId, session],
    queryFn: () => fetchBotOpenOrders(userId, session),
    enabled: !!userId && !!session,
    refetchInterval: 10_000,
  })
}

export function useBotTrades(userId: string) {
  return useQuery({
    queryKey: ['bot-trades', userId],
    queryFn: () => fetchBotTrades(userId),
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useBotAccount(userId: string, session: string) {
  return useQuery({
    queryKey: ['bot-account', userId, session],
    queryFn: () => fetchBotAccount(userId, session),
    enabled: !!userId && !!session,
    refetchInterval: 15_000,
  })
}

export function useCredentials(userId: string) {
  return useQuery<CredentialInfo[]>({
    queryKey: ['credentials', userId],
    queryFn: () => apiFetchCredentials(userId),
    enabled: !!userId,
  })
}

export function useCreateCredential() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: apiCreateCredential,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['credentials'] }),
  })
}

export function useDeleteCredential() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, userId }: { id: string; userId: string }) => apiDeleteCredential(id, userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['credentials'] }),
  })
}

export function useSupabaseOrders(userId: string) {
  return useQuery({
    queryKey: ['orders', userId],
    queryFn: async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data } = await supabase
        .from('sync_orders')
        .select('*')
        .eq('user_id', userId)
        .order('created_at', { ascending: false })
        .limit(50)
      return data ?? []
    },
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useSupabaseTrades(userId: string) {
  return useQuery({
    queryKey: ['trades', userId],
    queryFn: async () => {
      const { createClient } = await import('@/lib/supabase/client')
      const supabase = createClient()
      const { data } = await supabase
        .from('sync_trades')
        .select('*')
        .eq('user_id', userId)
        .order('created_at', { ascending: false })
        .limit(50)
      return data ?? []
    },
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export type { StrategyEntry, UserContainer, CredentialInfo, BacktestResult }
