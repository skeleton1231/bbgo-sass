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
  fetchMarketKlines,
  fetchBotList,
  fetchBotDetail,
  fetchMarketTicker,
  type TradingVolumeEntry,
  type StrategyEntry,
  type UserContainer,
  type UserContainersResponse,
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
  type MarketTicker,
  type Bot,
} from './manager'

// --- Strategy & container queries ---

export function useUserStrategies(userId: string) {
  return useQuery<UserContainersResponse>({
    queryKey: ['user-strategies', userId],
    queryFn: () => fetchUserStrategies(userId),
    enabled: !!userId,
    staleTime: 15_000,
    refetchInterval: 15_000,
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
    mutationFn: ({ userId, mode }: { userId: string; mode?: 'live' | 'paper' }) =>
      apiStartUser(userId, mode),
    onSuccess: (_data, { userId }) => qc.invalidateQueries({ queryKey: ['user-strategies', userId] }),
  })
}

export function useStopUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, mode }: { userId: string; mode?: 'live' | 'paper' }) =>
      apiStopUser(userId, mode),
    onSuccess: (_data, { userId }) => qc.invalidateQueries({ queryKey: ['user-strategies', userId] }),
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
    refetchInterval: (query) => {
      const jobs = query.state.data?.jobs ?? []
      const hasActive = jobs.some(j => j.status === 'pending' || j.status === 'downloading' || j.status === 'running')
      return hasActive ? 5_000 : 60_000
    },
  })
}

// --- Bot list & detail ---

export function useBotList(userId: string, mode?: 'live' | 'paper') {
  return useQuery<{ bots: Bot[] }>({
    queryKey: ['bot-list', userId, mode],
    queryFn: () => fetchBotList(userId, mode),
    enabled: !!userId,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotDetail(userId: string, botId: string | null) {
  return useQuery<Bot>({
    queryKey: ['bot-detail', userId, botId],
    queryFn: () => fetchBotDetail(userId, botId!),
    enabled: !!userId && !!botId,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

// --- Bot data queries (real-time from bbgo container) ---

export function useBotPing(userId: string, mode?: 'live' | 'paper') {
  return useQuery<{ status: string }>({
    queryKey: ['bot-ping', userId, mode],
    queryFn: () => fetchBotPing(userId, mode),
    enabled: !!userId,
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useBotSessions(userId: string, mode?: 'live' | 'paper') {
  return useQuery<{ sessions: BBGoSession[] }>({
    queryKey: ['bot-sessions', userId, mode],
    queryFn: () => fetchBotSessions(userId, mode),
    enabled: !!userId,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotSessionTrades(userId: string, session: string, mode?: 'live' | 'paper') {
  return useQuery<{ trades: BBGoTrade[] }>({
    queryKey: ['bot-session-trades', userId, session, mode],
    queryFn: () => fetchBotSessionTrades(userId, session, mode),
    enabled: !!userId && !!session,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotOpenOrders(userId: string, session: string, mode?: 'live' | 'paper') {
  return useQuery<{ orders: BBGoOrder[] }>({
    queryKey: ['bot-orders', userId, session, mode],
    queryFn: () => fetchBotOpenOrders(userId, session, mode),
    enabled: !!userId && !!session,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotSessionBalances(userId: string, session: string, mode?: 'live' | 'paper') {
  return useQuery<{ balances: Record<string, BBGoBalance> }>({
    queryKey: ['bot-balances', userId, session, mode],
    queryFn: () => fetchBotSessionBalances(userId, session, mode),
    enabled: !!userId && !!session,
    staleTime: 20_000,
    refetchInterval: 20_000,
  })
}

export function useBotSessionSymbols(userId: string, session: string, mode?: 'live' | 'paper') {
  return useQuery<{ symbols: string[] }>({
    queryKey: ['bot-symbols', userId, session, mode],
    queryFn: () => fetchBotSessionSymbols(userId, session, mode),
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

export function useMarketTicker(exchange: string, symbol: string) {
  return useQuery<{ ticker: MarketTicker }>({
    queryKey: ['market-ticker', exchange, symbol],
    queryFn: () => fetchMarketTicker(exchange, symbol),
    enabled: !!exchange && !!symbol,
    staleTime: 30_000,
  })
}

export function useMarketKlines(exchange: string, symbol: string, interval?: string) {
  return useQuery<{ klines: Array<{ time: number; open: string; high: string; low: string; close: string; volume: string; closed: boolean }> }>({
    queryKey: ['market-klines', exchange, symbol, interval],
    queryFn: () => fetchMarketKlines(exchange, symbol, interval),
    enabled: !!exchange && !!symbol,
    staleTime: 60_000,
  })
}

export function useBotTrades(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper') {
  return useQuery<{ trades: BBGoTrade[] }>({
    queryKey: ['bot-trades', userId, exchange, symbol, mode],
    queryFn: () => fetchBotTrades(userId, exchange, symbol, undefined, mode),
    enabled: !!userId,
    staleTime: 20_000,
    refetchInterval: 20_000,
  })
}

export function useBotClosedOrders(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper') {
  return useQuery<{ orders: BBGoOrder[] }>({
    queryKey: ['bot-closed-orders', userId, exchange, symbol, mode],
    queryFn: () => fetchBotClosedOrders(userId, exchange, symbol, undefined, mode),
    enabled: !!userId,
    staleTime: 20_000,
    refetchInterval: 20_000,
  })
}

export function useBotTradingVolume(userId: string, period?: string, mode?: 'live' | 'paper') {
  return useQuery<{ tradingVolumes: TradingVolumeEntry[] }>({
    queryKey: ['bot-trading-volume', userId, period, mode],
    queryFn: () => fetchBotTradingVolume(userId, period, undefined, mode),
    enabled: !!userId,
    staleTime: 60_000,
    refetchInterval: 60_000,
  })
}

export function useBotAssets(userId: string, mode?: 'live' | 'paper') {
  return useQuery<{ assets: Record<string, BBGoAsset> }>({
    queryKey: ['bot-assets', userId, mode],
    queryFn: () => fetchBotAssets(userId, mode),
    enabled: !!userId,
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useBotStrategiesState(userId: string, mode?: 'live' | 'paper') {
  return useQuery<{ strategies: BBGoStrategyState[] }>({
    queryKey: ['bot-strategies-state', userId, mode],
    queryFn: () => fetchBotStrategies(userId, mode),
    enabled: !!userId,
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useContainerLogs(userId: string, tail?: string, mode?: 'live' | 'paper') {
  return useQuery<{ logs: string }>({
    queryKey: ['container-logs', userId, tail, mode],
    queryFn: () => fetchContainerLogs(userId, tail, mode),
    enabled: !!userId,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotPnL(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper') {
  return useQuery<PnLReport>({
    queryKey: ['bot-pnl', userId, exchange, symbol, mode],
    queryFn: () => fetchBotPnL(userId, exchange, symbol, mode),
    enabled: !!userId,
    staleTime: 30_000,
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
  Bot,
  StrategyEntry,
  UserContainer,
  UserContainersResponse,
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
