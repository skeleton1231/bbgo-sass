import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchUserStrategies,
  createStrategy as apiCreateStrategy,
  deleteStrategy as apiDeleteStrategy,
  startUser as apiStartUser,
  stopUser as apiStopUser,
  startInstance as apiStartInstance,
  stopInstance as apiStopInstance,
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
  fetchBotTradeMarkers,
  type TradingVolumeEntry,
  type TradeMarkersResponse,
  type InstanceInfo,
  type InstanceListResponse,
  type BacktestResult,
  type BacktestJob,
  type BacktestReport,
  type BacktestSymbolReport,
  type SubmitBacktestResponse,
  type CredentialInfo,
  type BBGoSession,
  type BBGoTrade,
  type BBGoOrder,
  type BBGoBalance,
  type BBGoAsset,
  type BBGoStrategyState,
  type PnLReport,
  type DailyPnl,
  type PnlCurvePoint,
  type MarketTicker,
  type Bot,
} from './manager'

// --- Strategy & container queries ---

export function useUserStrategies(userId: string) {
  return useQuery<InstanceListResponse>({
    queryKey: ['user-strategies', userId],
    queryFn: () => fetchUserStrategies(userId),
    enabled: !!userId,
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

function invalidateUserQueries(qc: ReturnType<typeof useQueryClient>, userId: string) {
  qc.invalidateQueries({ queryKey: ['user-strategies', userId] })
  qc.invalidateQueries({ queryKey: ['bot-list', userId] })
  qc.invalidateQueries({ queryKey: ['bot-detail', userId] })
}

export function useCreateStrategy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, ...data }: { userId: string } & Parameters<typeof apiCreateStrategy>[1]) =>
      apiCreateStrategy(userId, data),
    onSuccess: (_data, variables) => invalidateUserQueries(qc, variables.userId),
  })
}

export function useDeleteStrategy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, strategyId }: { userId: string; strategyId: string }) =>
      apiDeleteStrategy(userId, strategyId),
    onSuccess: (_data, variables) => invalidateUserQueries(qc, variables.userId),
  })
}

export function useStartUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, mode }: { userId: string; mode?: 'live' | 'paper' }) =>
      apiStartUser(userId, mode),
    onSuccess: (_data, { userId }) => invalidateUserQueries(qc, userId),
  })
}

export function useStopUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, mode }: { userId: string; mode?: 'live' | 'paper' }) =>
      apiStopUser(userId, mode),
    onSuccess: (_data, { userId }) => invalidateUserQueries(qc, userId),
  })
}

export function useStartInstance() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, instanceId }: { userId: string; instanceId: string }) =>
      apiStartInstance(userId, instanceId),
    onSuccess: (_data, { userId }) => invalidateUserQueries(qc, userId),
  })
}

export function useStopInstance() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, instanceId }: { userId: string; instanceId: string }) =>
      apiStopInstance(userId, instanceId),
    onSuccess: (_data, { userId }) => invalidateUserQueries(qc, userId),
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

export function useBotPing(userId: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategyInstanceID?: string) {
  return useQuery<{ status: string }>({
    queryKey: ['bot-ping', userId, mode, strategyInstanceID],
    queryFn: () => fetchBotPing(userId, mode, strategyInstanceID),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useBotSessions(userId: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategyInstanceID?: string) {
  return useQuery<{ sessions: BBGoSession[] }>({
    queryKey: ['bot-sessions', userId, mode, strategyInstanceID],
    queryFn: () => fetchBotSessions(userId, mode, strategyInstanceID),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotSessionTrades(userId: string, session: string, mode?: 'live' | 'paper', containerRunning?: boolean) {
  return useQuery<{ trades: BBGoTrade[] }>({
    queryKey: ['bot-session-trades', userId, session, mode],
    queryFn: () => fetchBotSessionTrades(userId, session, mode),
    enabled: !!userId && !!session && (containerRunning ?? false),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotOpenOrders(userId: string, session: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategyInstanceID?: string) {
  return useQuery<{ orders: BBGoOrder[] }>({
    queryKey: ['bot-orders', userId, session, mode, strategyInstanceID],
    queryFn: () => fetchBotOpenOrders(userId, session, mode, strategyInstanceID),
    enabled: !!userId && !!session && (containerRunning ?? false),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotSessionBalances(userId: string, session: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategyInstanceID?: string) {
  return useQuery<{ balances: Record<string, BBGoBalance> }>({
    queryKey: ['bot-balances', userId, session, mode, strategyInstanceID],
    queryFn: () => fetchBotSessionBalances(userId, session, mode, strategyInstanceID),
    enabled: !!userId && !!session && (containerRunning ?? false),
    staleTime: 20_000,
    refetchInterval: 20_000,
  })
}

export function useBotSessionSymbols(userId: string, session: string, mode?: 'live' | 'paper', containerRunning?: boolean) {
  return useQuery<{ symbols: string[] }>({
    queryKey: ['bot-symbols', userId, session, mode],
    queryFn: () => fetchBotSessionSymbols(userId, session, mode),
    enabled: !!userId && !!session && (containerRunning ?? false),
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

export function useBotTrades(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategyInstanceID?: string) {
  return useQuery<{ trades: BBGoTrade[] }>({
    queryKey: ['bot-trades', userId, exchange, symbol, mode, strategyInstanceID],
    queryFn: () => fetchBotTrades(userId, exchange, symbol, undefined, mode, { strategy: strategyInstanceID }),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 20_000,
    refetchInterval: 20_000,
  })
}

export function useBotClosedOrders(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper', containerRunning?: boolean) {
  return useQuery<{ orders: BBGoOrder[] }>({
    queryKey: ['bot-closed-orders', userId, exchange, symbol, mode],
    queryFn: () => fetchBotClosedOrders(userId, exchange, symbol, undefined, mode),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 20_000,
    refetchInterval: 20_000,
  })
}

export function useBotTradeMarkers(userId: string, symbol: string, opts?: { exchange?: string; since?: string; until?: string; limit?: number; mode?: 'live' | 'paper'; strategy?: string }, containerRunning?: boolean) {
  return useQuery<TradeMarkersResponse>({
    queryKey: ['bot-trade-markers', userId, symbol, opts],
    queryFn: () => fetchBotTradeMarkers(userId, symbol, { ...opts, mode: opts?.mode }),
    enabled: !!userId && !!symbol && (containerRunning ?? false),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotTradingVolume(userId: string, period?: string, mode?: 'live' | 'paper', containerRunning?: boolean) {
  return useQuery<{ tradingVolumes: TradingVolumeEntry[] }>({
    queryKey: ['bot-trading-volume', userId, period, mode],
    queryFn: () => fetchBotTradingVolume(userId, period, undefined, mode),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 60_000,
    refetchInterval: 60_000,
  })
}

export function useBotAssets(userId: string, mode?: 'live' | 'paper', containerRunning?: boolean) {
  return useQuery<{ assets: Record<string, BBGoAsset> }>({
    queryKey: ['bot-assets', userId, mode],
    queryFn: () => fetchBotAssets(userId, mode),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useBotStrategiesState(userId: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategyInstanceID?: string) {
  return useQuery<{ strategies: BBGoStrategyState[] }>({
    queryKey: ['bot-strategies-state', userId, mode, strategyInstanceID],
    queryFn: () => fetchBotStrategies(userId, mode, strategyInstanceID),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useContainerLogs(userId: string, tail?: string, mode?: 'live' | 'paper', containerRunning?: boolean) {
  return useQuery<{ logs: string }>({
    queryKey: ['container-logs', userId, tail, mode],
    queryFn: () => fetchContainerLogs(userId, tail, mode),
    enabled: !!userId && (containerRunning ?? false),
    staleTime: 15_000,
    refetchInterval: 15_000,
  })
}

export function useBotPnL(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper', containerRunning?: boolean, strategy?: string) {
  return useQuery<PnLReport>({
    queryKey: ['bot-pnl', userId, exchange, symbol, mode, strategy],
    queryFn: async (): Promise<PnLReport> => {
      const raw = await fetchBotPnL(userId, exchange, symbol, mode, strategy)
      return {
        totalRealizedPnl: raw.totalRealizedPnl ?? 0,
        totalUnrealizedPnl: raw.totalUnrealizedPnl ?? 0,
        totalFees: raw.totalFees ?? 0,
        totalTrades: raw.totalTrades ?? 0,
        winningTrades: raw.winningTrades ?? 0,
        losingTrades: raw.losingTrades ?? 0,
        winRate: raw.winRate ?? 0,
        symbols: (raw.symbols ?? []).map((sym) => ({
          symbol: sym.symbol ?? '',
          realizedPnl: sym.realizedPnl ?? 0,
          totalBuys: sym.totalBuys ?? 0,
          totalSells: sym.totalSells ?? 0,
          buyVolume: sym.buyVolume ?? 0,
          sellVolume: sym.sellVolume ?? 0,
          totalFees: sym.totalFees ?? 0,
          tradeCount: sym.tradeCount ?? 0,
          winningTrades: sym.winningTrades ?? 0,
          losingTrades: sym.losingTrades ?? 0,
          avgBuyPrice: sym.avgBuyPrice ?? 0,
          avgSellPrice: sym.avgSellPrice ?? 0,
          openPosition: sym.openPosition ?? 0,
          openPositionCost: sym.openPositionCost ?? 0,
          unrealizedPnl: sym.unrealizedPnl ?? 0,
          currentPrice: sym.currentPrice ?? 0,
        })),
        dailyBreakdown: raw.dailyBreakdown ?? [],
        pnlCurve: raw.pnlCurve ?? [],
      }
    },
    enabled: !!userId && (containerRunning ?? false),
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
  InstanceInfo,
  InstanceListResponse,
  CredentialInfo,
  BacktestResult,
  BacktestJob,
  BacktestReport,
  BacktestSymbolReport,
  SubmitBacktestResponse,
  BBGoSession,
  BBGoTrade,
  BBGoOrder,
  BBGoBalance,
  BBGoAsset,
  BBGoStrategyState,
  PnLReport,
  DailyPnl,
  PnlCurvePoint,
  TradingVolumeEntry,
  TradeMarkersResponse,
}
