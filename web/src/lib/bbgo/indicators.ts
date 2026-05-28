import type { Time } from 'lightweight-charts'

export interface IndicatorPoint {
  time: Time
  value: number
}

export function computeSMA(
  closes: Array<{ time: Time; close: number }>,
  period: number
): IndicatorPoint[] {
  if (closes.length < period) return []
  const result: IndicatorPoint[] = []
  for (let i = period - 1; i < closes.length; i++) {
    let sum = 0
    for (let j = i - period + 1; j <= i; j++) {
      sum += closes[j]!.close
    }
    result.push({ time: closes[i]!.time, value: sum / period })
  }
  return result
}

export function computeEMA(
  closes: Array<{ time: Time; close: number }>,
  period: number
): IndicatorPoint[] {
  if (closes.length < period) return []
  const k = 2 / (period + 1)
  const result: IndicatorPoint[] = []

  let sum = 0
  for (let i = 0; i < period; i++) {
    sum += closes[i]!.close
  }
  let ema = sum / period
  result.push({ time: closes[period - 1]!.time, value: ema })

  for (let i = period; i < closes.length; i++) {
    ema = closes[i]!.close * k + ema * (1 - k)
    result.push({ time: closes[i]!.time, value: ema })
  }
  return result
}

export function computeBollingerBands(
  closes: Array<{ time: Time; close: number }>,
  period: number,
  multiplier: number
): { upper: IndicatorPoint[]; middle: IndicatorPoint[]; lower: IndicatorPoint[] } {
  if (closes.length < period) return { upper: [], middle: [], lower: [] }

  const middle = computeSMA(closes, period)
  const upper: IndicatorPoint[] = []
  const lower: IndicatorPoint[] = []

  for (let i = period - 1; i < closes.length; i++) {
    const slice = closes.slice(i - period + 1, i + 1)
    const mean = middle[i - period + 1]!.value
    let variance = 0
    for (const s of slice) {
      variance += (s.close - mean) ** 2
    }
    const stdDev = Math.sqrt(variance / period)
    const idx = i - period + 1
    upper.push({ time: middle[idx]!.time, value: mean + multiplier * stdDev })
    lower.push({ time: middle[idx]!.time, value: mean - multiplier * stdDev })
  }

  return { upper, middle, lower }
}

export type IndicatorConfig = {
  id: string
  name: string
  type: 'sma' | 'ema' | 'bollinger'
  period: number
  color: string
  enabled: boolean
}

export const DEFAULT_INDICATORS: IndicatorConfig[] = [
  { id: 'sma20', name: 'SMA(20)', type: 'sma', period: 20, color: '#f59e0b', enabled: false },
  { id: 'ema12', name: 'EMA(12)', type: 'ema', period: 12, color: '#8b5cf6', enabled: false },
  { id: 'ema26', name: 'EMA(26)', type: 'ema', period: 26, color: '#ec4899', enabled: false },
  { id: 'boll20', name: 'BOLL(20,2)', type: 'bollinger', period: 20, color: '#06b6d4', enabled: false },
]
