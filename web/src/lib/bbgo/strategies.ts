export type FieldType = 'number' | 'text' | 'boolean' | 'select' | 'group'

export interface FieldDef {
  key: string
  label: string
  type: FieldType
  default: string | number | boolean
  options?: string[]
  min?: number
  max?: number
  step?: number
  required?: boolean
  description?: string
  group?: string
}

export interface SessionRole {
  name: string
  label: string
  futures: boolean
}

export interface FuturesConfig {
  leverage?: number
  marginType?: 'cross' | 'isolated'
}

/**
 * Universal risk parameters enforced by bbgo's UniversalRiskController
 * (auto-bound by GeneralOrderExecutor.Bind when BBGO_UNIVERSAL_RISK_* env
 * vars are present). Applies to ANY strategy — not gated by futures flag.
 *
 * All fields are optional; zero/undefined means "unset" (skipped at emission).
 */
export interface RiskConfig {
  stopLossPrice?: number
  takeProfitPrice?: number
  roiStopLoss?: number
  roiTakeProfit?: number
  trailingActivation?: number
  trailingCallback?: number
  maxPositionQty?: number
}

export interface StrategySchema {
  id: string
  label: string
  description: string
  category: 'grid' | 'maker' | 'trend' | 'mean-reversion' | 'dca' | 'volatility' | 'other' | 'indicator' | 'cross-exchange' | 'utility'
  supportedExchanges: string[]
  fields: FieldDef[]
  crossExchange?: boolean
  sessionRoles?: SessionRole[]
  liveOnly?: boolean
  requiresFutures?: boolean
}

export function registryRowToSchema(row: {
  id: string
  display_name: string
  description: string | null
  category: string
  exchanges: unknown
  live_only: boolean | null
  cross_exchange: boolean | null
  fields: unknown
  session_roles: unknown
  sort_order: number | null
  requires_futures: boolean | null
}): StrategySchema {
  return {
    id: row.id,
    label: row.display_name,
    description: row.description ?? '',
    category: row.category as StrategySchema['category'],
    supportedExchanges: Array.isArray(row.exchanges) ? row.exchanges as string[] : [],
    fields: Array.isArray(row.fields) ? row.fields as FieldDef[] : [],
    liveOnly: row.live_only ?? false,
    crossExchange: row.cross_exchange ?? false,
    sessionRoles: Array.isArray(row.session_roles) ? row.session_roles as SessionRole[] : undefined,
    requiresFutures: row.requires_futures ?? false,
  }
}

export function getStrategySchema(id: string, registry?: StrategySchema[]): StrategySchema | undefined {
  return (registry ?? []).find((s) => s.id === id)
}

export function getStrategyDefaults(id: string, registry?: StrategySchema[]): Record<string, unknown> {
  const schema = getStrategySchema(id, registry)
  if (!schema) return {}
  const defaults: Record<string, unknown> = {}
  for (const field of schema.fields) {
    setNestedValue(defaults, field.key, field.default)
  }
  return defaults
}

function setNestedValue(obj: Record<string, unknown>, path: string, value: unknown) {
  const keys = path.split('.')
  let current: Record<string, unknown> = obj
  for (let i = 0; i < keys.length - 1; i++) {
    if (!(keys[i]! in current) || typeof current[keys[i]!] !== 'object' || current[keys[i]!] === null) {
      current[keys[i]!] = {}
    }
    current = current[keys[i]!] as Record<string, unknown>
  }
  current[keys[keys.length - 1]!] = value
}

export function nestConfig(config: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(config)) {
    if (value === undefined || value === null || value === '') continue
    setNestedValue(result, key, value)
  }
  return result
}

export function ensureTypes(schema: StrategySchema | undefined, config: Record<string, unknown>): Record<string, unknown> {
  if (!schema) return config
  const result = { ...config }
  for (const field of schema.fields) {
    const v = result[field.key]
    if (field.type === 'number') {
      if (v === '' || v === undefined || v === null) {
        delete result[field.key]
        continue
      }
      const num = Number(v)
      if (Number.isFinite(num)) result[field.key] = num
    } else if (field.type === 'boolean') {
      if (typeof v === 'string') {
        result[field.key] = v === 'true'
      } else if (typeof v !== 'boolean') {
        result[field.key] = Boolean(v)
      }
    }
  }
  return result
}

export function getAllStrategies(registry?: StrategySchema[]): { id: string; label: string; description: string; category: string }[] {
  return (registry ?? []).map((s) => ({ id: s.id, label: s.label, description: s.description, category: s.category }))
}

export function getStrategiesByCategory(opts?: { excludeLiveOnly?: boolean; excludeCrossExchange?: boolean }, registry?: StrategySchema[]): Record<string, { id: string; label: string; description: string }[]> {
  const grouped: Record<string, { id: string; label: string; description: string }[]> = {}
  for (const s of (registry ?? [])) {
    if (opts?.excludeLiveOnly && s.liveOnly) continue
    if (opts?.excludeCrossExchange && s.category === 'cross-exchange') continue
    const cat = s.category
    if (!grouped[cat]) grouped[cat] = []
    grouped[cat]!.push({ id: s.id, label: s.label, description: s.description })
  }
  return grouped
}
