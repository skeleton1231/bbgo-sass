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

export interface StrategyEntry {
  id: string
  name: string
  exchange: string
  strategy: string
  config: Record<string, unknown>
  mode: 'live' | 'paper'
}

export interface UserContainer {
  user_id: string
  status: 'running' | 'stopped' | 'error'
  strategies: StrategyEntry[]
}

export interface BacktestResult {
  output: string
}

export function fetchUserStrategies(userId: string) {
  return request<UserContainer>(`/users/${userId}/strategies`)
}

export function createStrategy(userId: string, data: {
  name: string
  exchange: string
  strategy: string
  config: Record<string, unknown>
  mode: 'live' | 'paper'
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

export function startUser(userId: string) {
  return request<UserContainer>(`/users/${userId}/start`, { method: 'POST' })
}

export function stopUser(userId: string) {
  return request<{ status: string; user_id: string }>(`/users/${userId}/stop`, { method: 'POST' })
}

export function fetchUserStatus(userId: string) {
  return request<UserContainer>(`/users/${userId}/status`)
}

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

export function fetchBotSessions(userId: string) {
  return request<unknown[]>(`/bbgo/${userId}/sessions`)
}

export function fetchBotOpenOrders(userId: string, session: string) {
  return request<unknown[]>(`/bbgo/${userId}/sessions/${session}/open-orders`)
}

export function fetchBotTrades(userId: string) {
  return request<unknown[]>(`/bbgo/${userId}/trades`)
}

export function fetchBotAccount(userId: string, session: string) {
  return request<unknown>(`/bbgo/${userId}/sessions/${session}/account`)
}

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
