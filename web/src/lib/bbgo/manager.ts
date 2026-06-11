const PROXY_PREFIX = '/api/manager'

function buildSessionQS(mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (strategyInstanceID) params.set('instanceID', strategyInstanceID)
  return params.toString()
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const { headers: customHeaders, ...rest } = options ?? {}
  const res = await fetch(`${PROXY_PREFIX}${path}`, {
    ...rest,
    headers: { 'Content-Type': 'application/json', ...customHeaders },
  })

  if (res.status === 401) {
    const locale = window.location.pathname.split('/')[1] || ''
    sessionStorage.setItem('bbgo-auth-message', 'session_expired')
    setTimeout(() => { window.location.href = `/${locale}/login` }, 5000)
    throw new Error('Session expired')
  }

  if (res.status === 503) {
    throw new Error('Auth service unavailable — please check your network and try again')
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `Manager API error: ${res.status}`)
  }

  return res.json()
}

// --- Strategy Instance types ---

export interface SessionRoleConfig {
  name: string
  exchange: string
  envVarPrefix: string
  futures: boolean
}

export interface StrategyWarning {
  id: string
  message: string
  level: 'critical' | 'warning'
}

export interface InstanceInfo {
  instance_id: string
  user_id: string
  mode: 'live' | 'paper'
  strategy: string
  symbol: string
  exchange: string
  name: string
  status: 'running' | 'stopped' | 'error' | 'starting'
  warnings?: StrategyWarning[]
}

export interface InstanceListResponse {
  user_id: string
  instances: InstanceInfo[]
}

// --- Bot (strategy instance in web UI) ---

export interface Bot {
  id: string
  strategy: string
  symbol: string
  exchange: string
  name: string
  config?: Record<string, unknown>
  state?: Record<string, unknown>
  container_status: 'running' | 'stopped' | 'error' | 'starting'
  container_name?: string
  mode: 'live' | 'paper'
}

export interface BotListResponse {
  bots: Bot[]
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
  report?: BacktestReport | null
  equity_curve?: string
  error?: string
  created_at: string
  started_at?: string
  completed_at?: string
  need_sync: boolean
  has_report?: boolean
}

export interface BacktestReport {
  startTime: string
  endTime: string
  sessions: string[]
  symbols: string[]
  initialEquityValue: number
  finalEquityValue: number
  totalProfit: number
  totalUnrealizedProfit: number
  totalGrossProfit: number
  totalGrossLoss: number
  symbolReports: BacktestSymbolReport[]
}

export interface BacktestSymbolReport {
  exchange: string
  symbol: string
  lastPrice: number
  startPrice: number
  tradeCount: number
  roundTurnCount: number
  totalNetProfit: number
  avgNetProfit: number
  grossProfit: number
  grossLoss: number
  prr: number
  percentProfitable: number
  maxDrawdown: number
  avgDrawdown: number
  maxProfit: number
  maxLoss: number
  avgProfit: number
  avgLoss: number
  winningCount: number
  losingCount: number
  maxLossStreak: number
  sharpeRatio: number
  sortinoRatio: number
  profitFactor: number
  winningRatio: number
  cagr: number
  calmar: number
  kelly: number
}

// --- BBGo bot data types (from bbgo container REST API) ---

export interface BBGoSession {
  name: string
  exchange: string
}

export type PositionAction =
  | 'OPEN' | 'ADD' | 'REDUCE' | 'CLOSE'
  | 'OPEN_LONG' | 'ADD_LONG' | 'REDUCE_LONG' | 'CLOSE_LONG'
  | 'OPEN_SHORT' | 'ADD_SHORT' | 'REDUCE_SHORT' | 'CLOSE_SHORT'
  | 'FLIP_LONG_TO_SHORT' | 'FLIP_SHORT_TO_LONG'

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
  strategyInstanceId?: string
  positionAction?: PositionAction
  netPosition?: number
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
  tag?: string
  strategyInstanceId?: string
  isFutures?: boolean
  positionAction?: PositionAction
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
  unrealizedPnl: number
  currentPrice: number
}

export interface DailyPnl {
  date: string
  pnl: number
}

export interface PnlCurvePoint {
  time: number
  value: number
}

export interface PnLReport {
  totalRealizedPnl: number
  totalUnrealizedPnl: number
  totalFees: number
  totalTrades: number
  winningTrades: number
  losingTrades: number
  winRate: number
  symbols: SymbolPnL[]
  dailyBreakdown: DailyPnl[]
  pnlCurve: PnlCurvePoint[]
}

// --- Strategy CRUD ---

export function fetchUserStrategies(userId: string) {
  return request<InstanceListResponse>(`/users/${userId}/strategies`)
}

export function createStrategy(userId: string, data: {
  name: string
  exchange: string
  strategy: string
  config: Record<string, unknown>
  mode: 'live' | 'paper'
  crossExchange?: boolean
  sessions?: SessionRoleConfig[]
  futuresConfig?: { leverage?: number; marginType?: string }
}) {
  return request<InstanceInfo>(`/users/${userId}/strategies`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteStrategy(userId: string, strategyId: string) {
  return request<{ status: string; instance_id: string; user_id: string; mode: string }>(`/users/${userId}/strategies/${strategyId}`, {
    method: 'DELETE',
  })
}

// --- Bot list & detail ---

export function fetchBotList(userId: string, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  if (mode) params.set('mode', mode)
  const qs = params.toString()
  return request<BotListResponse>(`/users/${userId}/bots${qs ? `?${qs}` : ''}`)
}

export function fetchBotDetail(userId: string, botId: string) {
  return request<Bot>(`/users/${userId}/bots/${botId}`)
}

// --- User lifecycle ---

export function startUser(userId: string, mode: 'live' | 'paper' = 'live') {
  return request<{ status: string; user_id: string; mode: string }>(`/users/${userId}/start?mode=${mode}`, { method: 'POST' })
}

export function stopUser(userId: string, mode: 'live' | 'paper' = 'live') {
  return request<{ status: string; user_id: string; mode: string }>(`/users/${userId}/stop?mode=${mode}`, { method: 'POST' })
}

export function fetchUserStatus(userId: string) {
  return request<InstanceListResponse>(`/users/${userId}/status`)
}

// --- Instance start/stop ---

export function startInstance(userId: string, instanceId: string) {
  return request<InstanceInfo>(`/users/${userId}/instances/${encodeURIComponent(instanceId)}/start`, { method: 'POST' })
}

export function stopInstance(userId: string, instanceId: string) {
  return request<{ status: string; instance_id: string; user_id: string; mode: string }>(`/users/${userId}/instances/${encodeURIComponent(instanceId)}/stop`, { method: 'POST' })
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

export function fetchBotPing(userId: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (strategyInstanceID) params.set('instanceID', strategyInstanceID)
  return request<{ status: string }>(`/users/${userId}/bbgo/ping?${params}`)
}

export function fetchBotSessions(userId: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (strategyInstanceID) params.set('instanceID', strategyInstanceID)
  return request<{ sessions: BBGoSession[] }>(`/users/${userId}/bbgo/sessions?${params}`)
}

export function fetchBotSessionDetail(userId: string, session: string, mode?: 'live' | 'paper') {
  return request<{ session: BBGoSession }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}?mode=${mode ?? 'live'}`)
}

export function fetchBotSessionTrades(userId: string, session: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const qs = buildSessionQS(mode, strategyInstanceID)
  return request<{ trades: BBGoTrade[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/trades?${qs}`)
}

export function fetchBotOpenOrders(userId: string, session: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const qs = buildSessionQS(mode, strategyInstanceID)
  return request<{ orders: BBGoOrder[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/open-orders?${qs}`)
}

export function fetchBotSessionAccount(userId: string, session: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const qs = buildSessionQS(mode, strategyInstanceID)
  return request<{ account: unknown }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/account?${qs}`)
}

export function fetchBotSessionBalances(userId: string, session: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const qs = buildSessionQS(mode, strategyInstanceID)
  return request<{ balances: Record<string, BBGoBalance> }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/balances?${qs}`)
}

export function fetchBotSessionSymbols(userId: string, session: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const qs = buildSessionQS(mode, strategyInstanceID)
  return request<{ symbols: string[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/symbols?${qs}`)
}

export function fetchBotAssets(userId: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const qs = buildSessionQS(mode, strategyInstanceID)
  return request<{ assets: Record<string, BBGoAsset> }>(`/users/${userId}/bbgo/assets?${qs}`)
}

export function fetchBotStrategies(userId: string, mode?: 'live' | 'paper', strategyInstanceID?: string) {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (strategyInstanceID) params.set('instanceID', strategyInstanceID)
  return request<{ strategies: BBGoStrategyState[] }>(`/users/${userId}/bbgo/strategies?${params}`)
}

export interface TradeMarkersResponse {
  markers: Array<{
    time: number
    side: 'BUY' | 'SELL'
    price: number
    quantity: number
    positionAction: string
  }>
}

export function fetchBotTrades(userId: string, exchange?: string, symbol?: string, gid?: number, mode?: 'live' | 'paper', opts?: { since?: string; until?: string; limit?: number; ordering?: string; strategy?: string }) {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  if (gid) params.set('gid', String(gid))
  if (opts?.since) params.set('since', opts.since)
  if (opts?.until) params.set('until', opts.until)
  if (opts?.limit) params.set('limit', String(opts.limit))
  if (opts?.ordering) params.set('ordering', opts.ordering)
  if (opts?.strategy) params.set('strategy', opts.strategy)
  const qs = params.toString()
  return request<{ trades: BBGoTrade[] }>(`/users/${userId}/bbgo/trades?${qs}`)
}

export function fetchBotTradeMarkers(userId: string, symbol: string, opts?: { exchange?: string; since?: string; until?: string; limit?: number; mode?: 'live' | 'paper'; strategy?: string }) {
  const params = new URLSearchParams()
  params.set('symbol', symbol)
  if (opts?.mode) params.set('mode', opts.mode)
  if (opts?.exchange) params.set('exchange', opts.exchange)
  if (opts?.since) params.set('since', opts.since)
  if (opts?.until) params.set('until', opts.until)
  if (opts?.limit) params.set('limit', String(opts.limit))
  if (opts?.strategy) params.set('strategy', opts.strategy)
  const qs = params.toString()
  return request<TradeMarkersResponse>(`/users/${userId}/bbgo/trades/markers?${qs}`)
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

export function fetchBotPnL(userId: string, exchange?: string, symbol?: string, mode?: 'live' | 'paper', strategy?: string) {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  if (strategy) params.set('strategy', strategy)
  return request<PnLReport>(`/users/${userId}/bbgo/pnl?${params}`)
}

// --- Futures & Margin data ---

export interface FuturesPositionRisk {
  id: string
  exchange: string
  symbol: string
  position_side: string
  strategy_instance_id: string
  leverage: string
  entry_price: string
  mark_price: string
  liquidation_price: string
  break_even_price: string
  position_amount: string
  unrealized_pnl: string
  notional: string
  initial_margin: string
  maint_margin: string
  position_initial_margin: string
  open_order_initial_margin: string
  adl: string
  margin_asset: string
  updated_at: string
}

export interface MarginLoan {
  id: string
  exchange: string
  asset: string
  isolated_symbol: string
  principle: string
  transaction_id: number
  time: string
}

export interface MarginRepay {
  id: string
  exchange: string
  asset: string
  isolated_symbol: string
  principle: string
  transaction_id: number
  time: string
}

export interface MarginInterest {
  id: string
  exchange: string
  asset: string
  isolated_symbol: string
  principle: string
  interest: string
  interest_rate: string
  time: string
}

export interface MarginLiquidation {
  id: string
  exchange: string
  symbol: string
  side: string
  order_id: number
  price: string
  quantity: string
  average_price: string
  executed_quantity: string
  time_in_force: string
  is_isolated: boolean
  time: string
}

export interface MarginHistoryResponse {
  loans: MarginLoan[]
  repays: MarginRepay[]
  interests: MarginInterest[]
  liquidations: MarginLiquidation[]
}

export function fetchFuturesPositions(userId: string, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  return request<{ positions: FuturesPositionRisk[] }>(`/users/${userId}/bbgo/futures/positions?${params}`)
}

export function fetchMarginHistory(userId: string, mode?: 'live' | 'paper') {
  const params = new URLSearchParams()
  params.set('mode', mode ?? 'live')
  return request<MarginHistoryResponse>(`/users/${userId}/bbgo/margin/history?${params}`)
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

export function fetchMarketTicker(exchange: string, symbol: string, session?: string) {
  const params = new URLSearchParams({ symbol })
  if (session) params.set('session', session)
  return request<{ ticker: MarketTicker }>(`/markets/${encodeURIComponent(exchange)}/ticker?${params}`)
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

export function fetchMarketKlines(exchange: string, symbol: string, interval?: string, limit?: number, startTime?: number, endTime?: number, session?: string) {
  const params = new URLSearchParams({ symbol })
  if (interval) params.set('interval', interval)
  if (limit) params.set('limit', String(limit))
  if (startTime) params.set('start_time', String(startTime))
  if (endTime) params.set('end_time', String(endTime))
  if (session) params.set('session', session)
  return request<{ klines: MarketKline[] }>(`/markets/${encodeURIComponent(exchange)}/klines?${params}`)
}

// --- Credentials ---

export interface CredentialInfo {
  id: string
  user_id: string
  exchange: string
  is_testnet: boolean
  is_verified: boolean
  verify_error?: string
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
