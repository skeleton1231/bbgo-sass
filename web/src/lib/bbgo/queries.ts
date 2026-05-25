import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchUserStrategies,
  createStrategy as apiCreateStrategy,
  deleteStrategy as apiDeleteStrategy,
  startUser as apiStartUser,
  stopUser as apiStopUser,
  runBacktest as apiRunBacktest,
  submitBacktest as apiSubmitBacktest,
  getBacktestJob as apiGetBacktestJob,
  listBacktestJobs as apiListBacktestJobs,
  fetchCredentials as apiFetchCredentials,
  createCredential as apiCreateCredential,
  deleteCredential as apiDeleteCredential,
  fetchBotPing,
  fetchBotSessions,
  fetchBotSessionTrades,
  fetchBotOpenOrders,
  fetchBotSessionBalances,
  fetchBotSessionSymbols,
  fetchBotTrades,
  fetchBotClosedOrders,
  fetchBotTradingVolume,
  fetchBotAssets,
  fetchBotStrategies,
  fetchContainerLogs,
  fetchBotPnL,
  fetchMarketSymbols,
  type TradingVolumeEntry,
  type StrategyEntry,
  type UserContainer,
  type BacktestResult,
  type BacktestJob,
  type SubmitBacktestResponse,
  type CredentialInfo,
  type BBGoSession,
  type BBGoTrade,
  type BBGoOrder,
  type BBGoBalance,
  type BBGoAsset,
  type BBGoStrategyState,
  type PnLReport,
} from './manager'

// --- Strategy & container queries ---

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

export function useSubmitBacktest() {
  return useMutation<SubmitBacktestResponse, Error, Parameters<typeof apiSubmitBacktest>[0]>({
    mutationFn: apiSubmitBacktest,
  })
}

export function useBacktestJob(jobId: string | null) {
  return useQuery<BacktestJob>({
    queryKey: ['backtest-job', jobId],
    queryFn: () => apiGetBacktestJob(jobId!),
    enabled: !!jobId,
    refetchInterval: (query) => {
      const data = query.state.data
      if (data && (data.status === 'pending' || data.status === 'downloading' || data.status === 'running')) {
        return 2_000
      }
      return false
    },
  })
}

export function useBacktestJobs() {
  return useQuery<{ jobs: BacktestJob[] }>({
    queryKey: ['backtest-jobs'],
    queryFn: apiListBacktestJobs,
    refetchInterval: 10_000,
  })
}

// --- Bot data queries (real-time from bbgo container) ---

export function useBotPing(userId: string) {
  return useQuery<{ status: string }>({
    queryKey: ['bot-ping', userId],
    queryFn: () => fetchBotPing(userId),
    enabled: !!userId,
    refetchInterval: 30_000,
  })
}

export function useBotSessions(userId: string) {
  return useQuery<{ sessions: BBGoSession[] }>({
    queryKey: ['bot-sessions', userId],
    queryFn: () => fetchBotSessions(userId),
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useBotSessionTrades(userId: string, session: string) {
  return useQuery<{ trades: BBGoTrade[] }>({
    queryKey: ['bot-session-trades', userId, session],
    queryFn: () => fetchBotSessionTrades(userId, session),
    enabled: !!userId && !!session,
    refetchInterval: 10_000,
  })
}

export function useBotOpenOrders(userId: string, session: string) {
  return useQuery<{ orders: BBGoOrder[] }>({
    queryKey: ['bot-orders', userId, session],
    queryFn: () => fetchBotOpenOrders(userId, session),
    enabled: !!userId && !!session,
    refetchInterval: 10_000,
  })
}

export function useBotSessionBalances(userId: string, session: string) {
  return useQuery<{ balances: Record<string, BBGoBalance> }>({
    queryKey: ['bot-balances', userId, session],
    queryFn: () => fetchBotSessionBalances(userId, session),
    enabled: !!userId && !!session,
    refetchInterval: 15_000,
  })
}

export function useBotSessionSymbols(userId: string, session: string) {
  return useQuery<{ symbols: string[] }>({
    queryKey: ['bot-symbols', userId, session],
    queryFn: () => fetchBotSessionSymbols(userId, session),
    enabled: !!userId && !!session,
    refetchInterval: 30_000,
  })
}

export function useMarketSymbols(exchange: string) {
  return useQuery<{ symbols: string[] }>({
    queryKey: ['market-symbols', exchange],
    queryFn: () => fetchMarketSymbols(exchange),
    enabled: !!exchange,
    staleTime: 5 * 60_000,
  })
}

export function useBotTrades(userId: string, exchange?: string, symbol?: string) {
  return useQuery<{ trades: BBGoTrade[] }>({
    queryKey: ['bot-trades', userId, exchange, symbol],
    queryFn: () => fetchBotTrades(userId, exchange, symbol),
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useBotClosedOrders(userId: string, exchange?: string, symbol?: string) {
  return useQuery<{ orders: BBGoOrder[] }>({
    queryKey: ['bot-closed-orders', userId, exchange, symbol],
    queryFn: () => fetchBotClosedOrders(userId, exchange, symbol),
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useBotTradingVolume(userId: string, period?: string) {
  return useQuery<{ tradingVolumes: TradingVolumeEntry[] }>({
    queryKey: ['bot-trading-volume', userId, period],
    queryFn: () => fetchBotTradingVolume(userId, period),
    enabled: !!userId,
    refetchInterval: 60_000,
  })
}

export function useBotAssets(userId: string) {
  return useQuery<{ assets: Record<string, BBGoAsset> }>({
    queryKey: ['bot-assets', userId],
    queryFn: () => fetchBotAssets(userId),
    enabled: !!userId,
    refetchInterval: 30_000,
  })
}

export function useBotStrategiesState(userId: string) {
  return useQuery<{ strategies: BBGoStrategyState[] }>({
    queryKey: ['bot-strategies-state', userId],
    queryFn: () => fetchBotStrategies(userId),
    enabled: !!userId,
    refetchInterval: 30_000,
  })
}

export function useContainerLogs(userId: string, tail?: string) {
  return useQuery<{ logs: string }>({
    queryKey: ['container-logs', userId, tail],
    queryFn: () => fetchContainerLogs(userId, tail),
    enabled: !!userId,
    refetchInterval: 15_000,
  })
}

export function useBotPnL(userId: string, exchange?: string, symbol?: string) {
  return useQuery<PnLReport>({
    queryKey: ['bot-pnl', userId, exchange, symbol],
    queryFn: () => fetchBotPnL(userId, exchange, symbol),
    enabled: !!userId,
    refetchInterval: 30_000,
  })
}

// --- Credentials ---

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

export type {
  StrategyEntry,
  UserContainer,
  CredentialInfo,
  BacktestResult,
  BacktestJob,
  SubmitBacktestResponse,
  BBGoSession,
  BBGoTrade,
  BBGoOrder,
  BBGoBalance,
  BBGoAsset,
  BBGoStrategyState,
  PnLReport,
  TradingVolumeEntry,
}
