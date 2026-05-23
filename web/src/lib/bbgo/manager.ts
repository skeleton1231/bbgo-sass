const PROXY_PREFIX = '/api/manager'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${PROXY_PREFIX}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })

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
  status: 'running' | 'stopped' | 'error'
  strategies: StrategyEntry[]
}

export interface BacktestResult {
  output: string
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

// --- Strategy CRUD ---

export function fetchUserStrategies(userId: string) {
  return request<UserContainer>(`/users/${userId}/strategies`)
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

export function startUser(userId: string) {
  return request<UserContainer>(`/users/${userId}/start`, { method: 'POST' })
}

export function stopUser(userId: string) {
  return request<{ status: string; user_id: string }>(`/users/${userId}/stop`, { method: 'POST' })
}

export function fetchUserStatus(userId: string) {
  return request<UserContainer>(`/users/${userId}/status`)
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

// --- Bot data via Manager → bbgo container REST API ---

export function fetchBotPing(userId: string) {
  return request<{ status: string }>(`/users/${userId}/bbgo/ping`)
}

export function fetchBotSessions(userId: string) {
  return request<{ sessions: BBGoSession[] }>(`/users/${userId}/bbgo/sessions`)
}

export function fetchBotSessionDetail(userId: string, session: string) {
  return request<{ session: BBGoSession }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}`)
}

export function fetchBotSessionTrades(userId: string, session: string) {
  return request<{ trades: BBGoTrade[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/trades`)
}

export function fetchBotOpenOrders(userId: string, session: string) {
  return request<{ orders: BBGoOrder[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/open-orders`)
}

export function fetchBotSessionAccount(userId: string, session: string) {
  return request<{ account: unknown }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/account`)
}

export function fetchBotSessionBalances(userId: string, session: string) {
  return request<{ balances: Record<string, BBGoBalance> }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/balances`)
}

export function fetchBotSessionSymbols(userId: string, session: string) {
  return request<{ symbols: string[] }>(`/users/${userId}/bbgo/session/${encodeURIComponent(session)}/symbols`)
}

export function fetchBotAssets(userId: string) {
  return request<{ assets: Record<string, BBGoAsset> }>(`/users/${userId}/bbgo/assets`)
}

export function fetchBotStrategies(userId: string) {
  return request<{ strategies: BBGoStrategyState[] }>(`/users/${userId}/bbgo/strategies`)
}

export function fetchBotTrades(userId: string, exchange?: string, symbol?: string, gid?: number) {
  const params = new URLSearchParams()
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  if (gid) params.set('gid', String(gid))
  const qs = params.toString()
  return request<{ trades: BBGoTrade[] }>(`/users/${userId}/bbgo/trades${qs ? `?${qs}` : ''}`)
}

export function fetchBotClosedOrders(userId: string, exchange?: string, symbol?: string, gid?: number) {
  const params = new URLSearchParams()
  if (exchange) params.set('exchange', exchange)
  if (symbol) params.set('symbol', symbol)
  if (gid) params.set('gid', String(gid))
  const qs = params.toString()
  return request<{ orders: BBGoOrder[] }>(`/users/${userId}/bbgo/orders/closed${qs ? `?${qs}` : ''}`)
}

export function fetchBotTradingVolume(userId: string, period?: string, segment?: string) {
  const params = new URLSearchParams()
  if (period) params.set('period', period)
  if (segment) params.set('segment', segment)
  const qs = params.toString()
  return request<{ tradingVolumes: unknown }>(`/users/${userId}/bbgo/trading-volume${qs ? `?${qs}` : ''}`)
}

export function fetchContainerLogs(userId: string, tail?: string) {
  const params = new URLSearchParams()
  if (tail) params.set('tail', tail)
  const qs = params.toString()
  return request<{ logs: string }>(`/users/${userId}/logs${qs ? `?${qs}` : ''}`)
}

// --- Credentials ---

export interface CredentialInfo {
  id: string
  user_id: string
  exchange: string
  is_testnet: boolean
  is_verified: boolean
}

export function fetchCredentials(userId: string) {
  return request<CredentialInfo[]>(`/credentials?user_id=${userId}`)
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

export function deleteCredential(id: string, userId: string) {
  return request<{ status: string }>(`/credentials/${id}?user_id=${userId}`, {
    method: 'DELETE',
  })
}
