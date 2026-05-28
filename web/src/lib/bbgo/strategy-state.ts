import type { GridLine } from '@/components/chart/CandlestickChart'

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

export function extractStrategyStats(
  strategyState: Record<string, unknown>
): {
  symbol: string
  strategy: string
  upperPrice: number
  lowerPrice: number
  gridNumber: number
  quantity: number
  base: number
  quote: number
  instanceId: string
} | null {
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
    instanceId: state.Position?.strategyInstanceID ?? '',
  }
}
