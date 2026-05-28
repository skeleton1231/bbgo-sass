const PROXY_PREFIX = '/api/manager'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const { headers: customHeaders, ...rest } = options ?? {}
  const res = await fetch(`${PROXY_PREFIX}${path}`, {
    ...rest,
    headers: { 'Content-Type': 'application/json', ...customHeaders },
  })

  if (res.status === 401) {
    const locale = window.location.pathname.split('/')[1] || ''
    sessionStorage.setItem('bbgo-auth-message', 'session_expired')
    window.location.href = `/${locale}/login`
    throw new Error('Session expired')
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `Manager API error: ${res.status}`)
  }

  return res.json()
}

// --- Strategy & User Container types ---

export interface SessionRoleConfig {
  name: string
  exchange: string
  envVarPrefix: string
  futures: boolean
}

export interface StrategyEntry {
  id: string
  name: string
  exchange: string
  strategy: string
  config: Record<string, unknown>
  mode: 'live' | 'paper'
  crossExchange: boolean
  sessions: SessionRoleConfig[]
}

export interface UserContainer {
  user_id: string
  mode: 'live' | 'paper'
  status: 'running' | 'stopped' | 'error' | 'starting'
  strategies: StrategyEntry[]
}

export interface UserContainersResponse {
  user_id: string
  containers: Partial<Record<'live' | 'paper', UserContainer>>
}

export interface BacktestResult {
  output: string
}

export interface BacktestJob {
  id: string
  user_id: string
  strategy: string
  config: Record<string, unknown>
  exchange: string
  symbol: string
  start_time: string
  end_time: string
  status: 'pending' | 'downloading' | 'running' | 'completed' | 'failed'
  progress?: string
  output?: string
  error?: string
  created_at: string
  started_at?: string
  completed_at?: string
  need_sync: boolean
}

// --- BBGo bot data types (from bbgo container REST API) ---

export interface BBGoSession {
  name: string
  exchangeName: string
}

export interface BBGoTrade {
  gid: number
  id: number
  orderID: number
  orderUUID?: string
  exchange: string
  symbol: string
  side: 'BUY' | 'SELL'
  price: string
  quantity: string
  quoteQuantity: string
  isBuyer: boolean
  isMaker: boolean
  tradedAt: string
  fee: string
  feeCurrency: string
}

export interface BBGoOrder {
  gid: number
  orderID: number
  uuid?: string
  clientOrderID?: string
  exchange: string
  symbol: string
  side: 'BUY' | 'SELL'
  orderType: string
  price: string
  quantity: string
  executedQuantity: string
  status: string
  stopPrice?: string
  creationTime?: string
  isWorking?: boolean
}

export interface BBGoBalance {
  currency: string
  available: string
  locked: string
}

export interface BBGoAsset {
  currency: string
  total: string
  available: string
  lock: string
  borrowed: string
  netAsset: string
  netAssetInUSD: string
  netAssetInBTC: string
  priceInUSD: string
}

export interface BBGoStrategyState {
  strategy: string
  [key: string]: unknown
}

export interface SymbolPnL {
  symbol: string
  realizedPnl: number
  totalBuys: number
  totalSells: number
  buyVolume: number
  sellVolume: number
  totalFees: number
  tradeCount: number
  winningTrades: number
  losingTrades: number
  avgBuyPrice: number
  avgSellPrice: number
  openPosition: number
  openPositionCost: number
}

export interface PnLReport {
  totalRealizedPnl: number
  totalFees: number
  totalTrades: number
  winningTrades: number
  losingTrades: number
  winRate: number
  symbols: SymbolPnL[]
}

// --- Strategy CRUD ---

export function fetchUserStrategies(userId: string) {
  return request<UserContainersResponse>(`/users/${userId}/strategies`)
}

export function createStrategy(userId: string, data: {
  name: string
  exchange: string
  strategy: string
  config: Record<string, unknown>
  mode: 'live' | 'paper'
  crossExchange?: boolean
  sessions?: SessionRoleConfig[]
}) {
  return request<UserContainer>(`/users/${userId}/strategies`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteStrategy(userId: string, strategyId: string) {
  return request<UserContainer>(`/users/${userId}/strategies/${strategyId}`, {
    method: 'DELETE',
  })
}

// --- User lifecycle ---

export function startUser(userId: string, mode: 'live' | 'paper' = 'live') {
  return request<UserContainer>(`/users/${userId}/start?mode=${mode}`, { method: 'POST' })
}

export function stopUser(userId: string, mode: 'live' | 'paper' = 'live') {
  return request<{ status: string; user_id: string }>(`/users/${userId}/stop?mode=${mode}`, { method: 'POST' })
}

export function fetchUserStatus(userId: string) {
  return request<UserContainersResponse>(`/users/${userId}/status`)
}

// --- Backtest ---

export function runBacktest(data: {
  strategy: string
  config: Record<string, unknown>
  start_time?: string
  end_time?: string
}) {
  return request<BacktestResult>('/backtest', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

// --- Async Backtest ---

export interface SubmitBacktestResponse {
  job_id: string
  status: string
  need_sync: boolean
}

export function submitBacktest(data: {
  strategy: string
  config: Record<string, unknown>
  exchange?: string
  symbol?: string
  start_time?: string
  end_time?: string
}) {
  return request<SubmitBacktestResponse>('/backtest/submit', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getBacktestJob(jobId: string) {
  return request<BacktestJob>(`/backtest/jobs/${jobId}`)
}

export function listBacktestJobs() {
  return request<{ jobs: BacktestJob[] }>('/backtest/jobs')
}

// --- Bot data via Manager → bbgo container REST API ---

export function fetchBotPing(userId: string, mode?: 'live' | 'paper') {
  return request<{ status: string }>(`/users/${userId}/bbgo/ping?mode=${mode ?? 'live'}`)
}

export function fetchBotSessions(userId: string, mode?: 'live' | 'paper') {
  return request<{ sessions: BBGoSession[] }>(`/users/${userId}/bbgo/sessions?mode=${mode ?? 'live'}`)
}

export function fetchBotSessionDetail(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ session: BBGoSession }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}?mode=${mode ?? 'live'}`)
}

export function fetchBotSessionTrades(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ trades: BBGoTrade[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/trades?mode=${mode ?? 'live'}`)
}

export function fetchBotOpenOrders(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ orders: BBGoOrder[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/open-orders?mode=${mode ?? 'live'}`)
}

export function fetchBotSessionAccount(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ account: unknown }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/account?mode=${mode ?? 'live'}`)
}

export function fetchBotSessionBalances(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ balances: Record<string, BBGoBalance> }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/balances?mode=${mode ?? 'live'}`)
}

export function fetchBotSessionSymbols(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ symbols: string[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/symbols?mode=${mode ?? 'live'}`)
}

export function fetchBotAssets(userId: string, mode?: 'live' | 'paper') {
  return request<{ assets: Record<string, BBGoAsset> }>(`/users/${userId}/bbgo/assets?mode=${mode ?? 'live'}`)
}

export function fetchBotStrategies(userId: string, mode?: 'live' | 'paper') {
  return request<{ strategies: BBGoStrategyState[] }>(`/users/${userId}/bbgo/strategies?mode=${mode ?? 'live'}`)
}

export function fetchBotTrades(userId: string, exchange?: string, symbol?: string, gid?: number, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  if (gid) params.set('gid', String(gid))
  const qs = params.toString()
  return request<{ trades: BBGoTrade[] }>(`/users/${userId}/bbgo/trades?${qs}`)
}

export function fetchBotClosedOrders(userId: string, exchange?: string, symbol?: string, gid?: number, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  if (gid) params.set('gid', String(gid))
  const qs = params.toString()
  return request<{ orders: BBGoOrder[] }>(`/users/${userId}/bbgo/orders/closed?${qs}`)
}

export interface TradingVolumeEntry {
  year: number
  month?: number
  day?: number
  time?: string
  exchange?: string
  symbol?: string
  quoteVolume: number
}

export function fetchBotTradingVolume(userId: string, period?: string, segment?: string, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (period) params.set('period', period)
  if (segment) params.set('segment', segment)
  return request<{ tradingVolumes: TradingVolumeEntry[] }>(`/users/${userId}/bbgo/trading-volume?${params}`)
}

export function fetchContainerLogs(userId: string, tail?: string, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (tail) params.set('tail', tail)
  return request<{ logs: string }>(`/users/${userId}/logs?${params}`)
}

export function fetchBotPnL(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  return request<PnLReport>(`/users/${userId}/bbgo/pnl?${params}`)
}

// --- Market data (from shared marketdata service via Manager) ---

export function fetchMarketSymbols(exchange: string) {
  return request<{ symbols: string[] }>(`/markets/${encodeURIComponent(exchange)}/symbols`)
}

export interface MarketTicker {
  symbol: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export function fetchMarketTicker(exchange: string, symbol: string) {
  return request<{ ticker: MarketTicker }>(`/markets/${encodeURIComponent(exchange)}/ticker?symbol=${encodeURIComponent(symbol)}`)
}

export interface MarketKline {
  time: number
  open: string
  high: string
  low: string
  close: string
  volume: string
  quoteVolume?: string
  closed: boolean
}

export function fetchMarketKlines(exchange: string, symbol: string, interval?: string, limit?: number) {
  const params = new URLSearchParams({ symbol })
  if (interval) params.set('interval', interval)
  if (limit) params.set('limit', String(limit))
  return request<{ klines: MarketKline[] }>(`/markets/${encodeURIComponent(exchange)}/klines?${params}`)
}

// --- Credentials ---

export interface CredentialInfo {
  id: string
  user_id: string
  exchange: string
  is_testnet: boolean
  is_verified: boolean
}

export function fetchCredentials(_userId: string) {
  return request<CredentialInfo[]>('/credentials')
}

export function createCredential(data: {
  exchange: string
  api_key: string
  api_secret: string
  passphrase?: string
  is_testnet?: boolean
}) {
  return request<CredentialInfo>('/credentials', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteCredential(id: string, _userId: string) {
  return request<{ status: string }>(`/credentials/${id}`, {
    method: 'DELETE',
  })
}

// --- Notifications ---

export interface NotificationConfigInfo {
  id: string
  type: 'telegram' | 'slack'
  enabled: boolean
  rules: {
    trade_events: boolean
    order_events: boolean
    container_health: boolean
  }
}

export function fetchNotificationConfigs() {
  return request<NotificationConfigInfo[]>('/notifications/config')
}

export function createNotificationConfig(data: {
  type: 'telegram' | 'slack'
  token?: string
  chat_id?: string
  webhook_url?: string
  rules: {
    trade_events: boolean
    order_events: boolean
    container_health: boolean
  }
}) {
  return request<NotificationConfigInfo>('/notifications/config', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteNotificationConfig(id: string) {
  return request<{ status: string }>(`/notifications/config/${id}`, {
    method: 'DELETE',
  })
}

export function testNotification() {
  return request<{ status: string }>('/notifications/test', { method: 'POST' })
}
