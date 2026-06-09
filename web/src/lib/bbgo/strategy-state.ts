import type { GridLine } from '@/components/chart/CandlestickChart'

export type StrategyCategory = 'grid' | 'maker' | 'trend' | 'mean-reversion' | 'dca' | 'volatility' | 'other' | 'indicator' | 'cross-exchange' | 'utility'

// Grid strategies have upperPrice, lowerPrice, gridNumber, quantity
const GRID_STRATEGIES = new Set(['grid2', 'grid', 'bollgrid', 'xhedgegrid'])

// Maker strategies have spread, quantity
const MAKER_STRATEGIES = new Set(['bollmaker', 'fmaker', 'fixedmaker', 'scmaker', 'audacitymaker', 'linregmaker', 'rsmaker'])

// DCA strategies have budget/budget quota, rounds
const DCA_STRATEGIES = new Set(['dca', 'dca2', 'dca3'])

// Trend-following strategies
const TREND_STRATEGIES = new Set(['supertrend', 'emacross', 'swing', 'trendtrader', 'elliottwave', 'drift', 'pivotshort'])

export function getStrategyCategory(strategyId: string): StrategyCategory {
  if (GRID_STRATEGIES.has(strategyId)) return 'grid'
  if (MAKER_STRATEGIES.has(strategyId)) return 'maker'
  if (DCA_STRATEGIES.has(strategyId)) return 'dca'
  if (TREND_STRATEGIES.has(strategyId)) return 'trend'
  return 'other'
}

interface GridStrategyState {
  symbol?: string
  upperPrice?: number
  lowerPrice?: number
  gridNumber?: number
  quantity?: number
  stopLossPrice?: number
  takeProfitPrice?: number
  Position?: {
    base?: number | string
    quote?: number | string
    averageCost?: number | string
    strategyInstanceID?: string
    openedAt?: string
  }
  GridProfitStats?: {
    profit?: number | string
    totalArbitrage?: number
    startTime?: string
  }
}

export function extractGridLines(
  strategyState: Record<string, unknown>,
  currentPrice?: number
): GridLine[] {
  const grid2 = strategyState['grid2'] as GridStrategyState | undefined
  if (!grid2 || !grid2.upperPrice || !grid2.lowerPrice || !grid2.gridNumber) return []

  const { upperPrice, lowerPrice, gridNumber } = grid2
  const step = (upperPrice - lowerPrice) / gridNumber
  const lines: GridLine[] = []

  for (let i = 0; i <= gridNumber; i++) {
    const price = lowerPrice + step * i
    const isBound = i === 0 || i === gridNumber
    const isNearCurrent = currentPrice ? Math.abs(price - currentPrice) < step * 0.5 : false
    lines.push({
      price,
      label: `${price.toFixed(price > 1000 ? 0 : 2)}`,
      color: isBound
        ? 'rgba(99, 102, 241, 0.6)'
        : isNearCurrent
          ? 'rgba(99, 102, 241, 0.45)'
          : 'rgba(99, 102, 241, 0.2)',
    })
  }

  return lines
}

// --- Legacy StrategyStats (kept for backward compat) ---

export interface StrategyStats {
  symbol: string
  strategy: string
  upperPrice: number
  lowerPrice: number
  gridNumber: number
  quantity: number
  base: number
  quote: number
  averageCost: number
  instanceId: string
  openedAt: string
  stopLossPrice: number
  takeProfitPrice: number
}

export function extractStrategyStats(
  strategyState: Record<string, unknown>
): StrategyStats | null {
  const strategy = strategyState['strategy'] as string | undefined
  if (!strategy) return null

  const state = strategyState[strategy] as GridStrategyState | undefined
  if (!state) return null

  return {
    symbol: state.symbol ?? '',
    strategy,
    upperPrice: state.upperPrice ?? 0,
    lowerPrice: state.lowerPrice ?? 0,
    gridNumber: state.gridNumber ?? 0,
    quantity: state.quantity ?? 0,
    base: typeof state.Position?.base === 'number' ? state.Position.base : parseFloat(String(state.Position?.base ?? '0')),
    quote: typeof state.Position?.quote === 'number' ? state.Position.quote : parseFloat(String(state.Position?.quote ?? '0')),
    averageCost: typeof state.Position?.averageCost === 'number' ? state.Position.averageCost : parseFloat(String(state.Position?.averageCost ?? '0')),
    instanceId: state.Position?.strategyInstanceID ?? '',
    openedAt: state.Position?.openedAt ?? '',
    stopLossPrice: state.stopLossPrice ?? 0,
    takeProfitPrice: state.takeProfitPrice ?? 0,
  }
}

// --- New: Strategy Detail Fields ---

export interface StrategyDetailField {
  key: string
  label: string
  value: string | number
  format?: 'price' | 'number' | 'percent' | 'quantity' | 'raw'
  color?: 'up' | 'down' | 'muted'
}

export interface StrategyDetails {
  strategy: string
  category: StrategyCategory
  fields: StrategyDetailField[]
}

function numVal(v: unknown): number {
  if (typeof v === 'number') return v
  if (typeof v === 'string') return parseFloat(v) || 0
  return 0
}

function getInnerState(strategyState: Record<string, unknown>): Record<string, unknown> | null {
  const strategy = strategyState['strategy'] as string | undefined
  if (!strategy) return null
  const inner = strategyState[strategy]
  if (!inner || typeof inner !== 'object') return null
  return inner as Record<string, unknown>
}

export function extractStrategyDetails(
  strategyState: Record<string, unknown>
): StrategyDetails | null {
  const strategy = strategyState['strategy'] as string | undefined
  if (!strategy) return null

  const inner = getInnerState(strategyState)
  if (!inner) return null

  const category = getStrategyCategory(strategy)
  const fields: StrategyDetailField[] = []

  const pos = inner['Position'] as Record<string, unknown> | undefined
  const profitStats = inner['ProfitStats'] as Record<string, unknown> | undefined
  const gridProfitStats = inner['GridProfitStats'] as Record<string, unknown> | undefined

  switch (category) {
    case 'grid':
      extractGridFields(inner, fields)
      break
    case 'maker':
      extractMakerFields(inner, fields)
      break
    case 'dca':
      extractDcaFields(inner, fields)
      break
    case 'trend':
      extractTrendFields(inner, strategy, fields)
      break
    default:
      extractDefaultFields(inner, fields)
      break
  }

  if (pos) {
    const base = numVal(pos.base)
    const quote = numVal(pos.quote)
    const avgCost = numVal(pos.averageCost)
    if (base !== 0 || avgCost !== 0) {
      fields.push(
        { key: 'pos.base', label: 'holding', value: Math.abs(base).toFixed(6), format: 'quantity' },
        { key: 'pos.avgCost', label: 'entryPrice', value: avgCost, format: 'price' },
      )
      if (quote !== 0) {
        fields.push({ key: 'pos.quote', label: 'invested', value: Math.abs(quote).toFixed(2), format: 'raw' })
      }
    }
  }

  if (gridProfitStats) {
    const profit = numVal(gridProfitStats.profit)
    if (profit !== 0) {
      fields.push({ key: 'gridProfit', label: 'gridProfit', value: profit, format: 'number', color: profit >= 0 ? 'up' : 'down' })
    }
    const totalArb = numVal(gridProfitStats.totalArbitrage)
    if (totalArb > 0) {
      fields.push({ key: 'totalArbitrage', label: 'totalArbitrage', value: totalArb, format: 'number' })
    }
  }

  if (profitStats) {
    const numTrades = numVal(profitStats.numTrades)
    if (numTrades > 0) {
      fields.push({ key: 'numTrades', label: 'numTrades', value: numTrades, format: 'number' })
    }
  }

  return { strategy, category, fields }
}

function extractGridFields(inner: Record<string, unknown>, fields: StrategyDetailField[]) {
  const upper = numVal(inner.upperPrice)
  const lower = numVal(inner.lowerPrice)
  if (upper > 0 || lower > 0) {
    fields.push({ key: 'range', label: 'range', value: `${lower.toLocaleString()}–${upper.toLocaleString()}`, format: 'raw' })
  }
  const gridNum = numVal(inner.gridNumber)
  if (gridNum > 0) {
    fields.push({ key: 'gridNumber', label: 'grids', value: gridNum, format: 'number' })
  }
  const qty = numVal(inner.quantity)
  if (qty > 0) {
    fields.push({ key: 'quantity', label: 'qtyPerGrid', value: qty, format: 'quantity' })
  }
  const sl = numVal(inner.stopLossPrice)
  if (sl > 0) {
    fields.push({ key: 'stopLoss', label: 'stopLoss', value: sl, format: 'price', color: 'down' })
  }
  const tp = numVal(inner.takeProfitPrice)
  if (tp > 0) {
    fields.push({ key: 'takeProfit', label: 'takeProfit', value: tp, format: 'price', color: 'up' })
  }
}

function extractMakerFields(inner: Record<string, unknown>, fields: StrategyDetailField[]) {
  const spread = numVal(inner.spread)
  if (spread > 0) {
    fields.push({ key: 'spread', label: 'spread', value: spread, format: 'number' })
  }
  const qty = numVal(inner.quantity)
  if (qty > 0) {
    fields.push({ key: 'quantity', label: 'quantity', value: qty, format: 'quantity' })
  }
  const interval = inner.interval as string | undefined
  if (interval) {
    fields.push({ key: 'interval', label: 'interval', value: interval, format: 'raw' })
  }
}

function extractDcaFields(inner: Record<string, unknown>, fields: StrategyDetailField[]) {
  const budget = numVal(inner.budget)
  const budgetQuota = numVal(inner.budgetQuota)
  const totalBudget = budget || budgetQuota
  if (totalBudget > 0) {
    fields.push({ key: 'budget', label: 'budget', value: totalBudget, format: 'price' })
  }
  const interval = inner.interval as string | undefined
  if (interval) {
    fields.push({ key: 'interval', label: 'interval', value: interval, format: 'raw' })
  }
  const qty = numVal(inner.quantity)
  if (qty > 0) {
    fields.push({ key: 'quantity', label: 'quantity', value: qty, format: 'quantity' })
  }
  const leverage = numVal(inner.leverage)
  if (leverage > 0) {
    fields.push({ key: 'leverage', label: 'leverage', value: `${leverage}x`, format: 'raw' })
  }
}

function extractTrendFields(inner: Record<string, unknown>, strategyId: string, fields: StrategyDetailField[]) {
  const interval = inner.interval as string | undefined
  if (interval) {
    fields.push({ key: 'interval', label: 'interval', value: interval, format: 'raw' })
  }
  const qty = numVal(inner.quantity)
  if (qty > 0) {
    fields.push({ key: 'quantity', label: 'quantity', value: qty, format: 'quantity' })
  }
  const leverage = numVal(inner.leverage)
  if (leverage > 0) {
    fields.push({ key: 'leverage', label: 'leverage', value: `${leverage}x`, format: 'raw' })
  }
  if (strategyId === 'supertrend' || strategyId === 'swing') {
    const maInterval = inner.movingAverageInterval as string | undefined
    if (maInterval) {
      fields.push({ key: 'maInterval', label: 'maInterval', value: maInterval, format: 'raw' })
    }
  }
  if (strategyId === 'emacross') {
    const fastLen = numVal(inner.fastLength)
    const slowLen = numVal(inner.slowLength)
    if (fastLen > 0) {
      fields.push({ key: 'fastLength', label: 'fastLength', value: fastLen, format: 'number' })
    }
    if (slowLen > 0) {
      fields.push({ key: 'slowLength', label: 'slowLength', value: slowLen, format: 'number' })
    }
  }
}

function extractDefaultFields(inner: Record<string, unknown>, fields: StrategyDetailField[]) {
  const skip = new Set(['symbol', 'Position', 'ProfitStats', 'TradeStats', 'GridProfitStats',
    'ProfitStatsTracker', 'strategyInstanceID', 'StrategyController'])
  for (const [key, val] of Object.entries(inner)) {
    if (skip.has(key)) continue
    if (val == null || typeof val === 'object') continue
    const num = typeof val === 'number' ? val : parseFloat(String(val))
    if (Number.isFinite(num) && num !== 0) {
      fields.push({ key, label: key, value: num, format: 'number' })
    }
  }
}
