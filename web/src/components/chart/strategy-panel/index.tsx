'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { extractBaseCurrency } from '@/lib/bbgo/fifo-pnl'
import type { StrategyDetails, StrategyDetailField } from '@/lib/bbgo/strategy-state'
import type { GridLine } from '@/components/chart/CandlestickChart'

export interface StrategyPanelProps {
  details: StrategyDetails
  currentPrice?: number
  gridLines: GridLine[]
  unrealizedPnlFromReport?: number
  supabasePosition?: {
    base: number
    averageCost: number
    quote: number
    symbol: string
    isClosed: boolean
  }
  unrealizedPnlPct?: number
}

export function StrategySidePanel(props: StrategyPanelProps) {
  const sp = useTranslations('Bots.chartSidePanel')
  const { details, currentPrice, gridLines, unrealizedPnlFromReport, supabasePosition, unrealizedPnlPct } = props
  const pnlPct = unrealizedPnlPct ?? 0
  const baseSymbol = extractBaseCurrency(supabasePosition?.symbol ?? '')

  const configFields = details.fields.filter((f) => !f.key.startsWith('pos.'))

  return (
    <div className="hidden lg:flex flex-col gap-2 w-52 shrink-0 text-xs">
      {/* Strategy config panel */}
      {configFields.length > 0 && (
        <div className="rounded-lg border bg-muted/30 p-3 space-y-2">
          <p className="font-medium text-sm">{details.strategy}</p>
          <div className="space-y-1.5 font-mono text-muted-foreground">
            {configFields.map((field) => (
              <FieldRow key={field.key} field={field} sp={sp} />
            ))}
          </div>
        </div>
      )}

      {/* Position panel */}
      {supabasePosition && !supabasePosition.isClosed && (
        <div className="rounded-lg border bg-muted/30 p-3 space-y-2">
          <p className="font-medium text-sm">{sp('position')}</p>
          <div className="space-y-1.5 font-mono text-muted-foreground">
            <div className="flex justify-between items-center">
              <span>{sp('holding')}</span>
              <span className="text-trade-up font-medium">
                {Math.abs(supabasePosition.base).toFixed(6)} {baseSymbol}
              </span>
            </div>
            {supabasePosition.averageCost > 0 && (
              <div className="flex justify-between">
                <span>{sp('entryPrice')}</span>
                <span className="text-foreground">${supabasePosition.averageCost.toLocaleString()}</span>
              </div>
            )}
            {Math.abs(supabasePosition.quote) > 0 && (
              <div className="flex justify-between">
                <span>{sp('invested')}</span>
                <span className="text-foreground">${Math.abs(supabasePosition.quote).toFixed(2)}</span>
              </div>
            )}
            {currentPrice && unrealizedPnlFromReport !== undefined && (
              <div className="flex justify-between items-center">
                <span>{sp('unrealized')}</span>
                <div className="text-right">
                  <span className={cn(
                    'font-medium',
                    unrealizedPnlFromReport >= 0 ? 'text-trade-up' : 'text-trade-down'
                  )}>
                    {unrealizedPnlFromReport >= 0 ? '+' : ''}{unrealizedPnlFromReport.toFixed(2)}
                  </span>
                  <span className={cn(
                    'ml-1 text-[10px]',
                    pnlPct >= 0 ? 'text-trade-up/70' : 'text-trade-down/70'
                  )}>
                    ({pnlPct >= 0 ? '+' : ''}{pnlPct.toFixed(2)}%)
                  </span>
                </div>
              </div>
            )}
            {currentPrice && (
              <div className="flex justify-between">
                <span>{sp('currentValue')}</span>
                <span className="text-foreground">${(currentPrice * Math.abs(supabasePosition.base)).toFixed(2)}</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Grid levels (only for grid strategies) */}
      {gridLines.length > 0 && (
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="font-medium mb-1.5">{sp('gridLevels')}</p>
          <div className="max-h-40 overflow-y-auto space-y-0.5 font-mono text-muted-foreground">
            {gridLines.slice(0, 12).map((g, i) => (
              <div key={i} className="flex justify-between">
                <span className="text-[10px]">{g.label}</span>
                <span className="w-1.5 h-1.5 rounded-full mt-1.5" style={{ backgroundColor: g.color }} />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function FieldRow({ field, sp }: { field: StrategyDetailField; sp: (key: string) => string }) {
  const label = tryTranslate(field.label, sp)
  const displayValue = formatFieldValue(field)

  return (
    <div className="flex justify-between">
      <span>{label}</span>
      <span className={cn(
        'text-foreground',
        field.color === 'up' && 'text-trade-up',
        field.color === 'down' && 'text-trade-down',
        field.color === 'muted' && 'text-muted-foreground',
      )}>
        {displayValue}
      </span>
    </div>
  )
}

function tryTranslate(label: string, sp: (key: string) => string): string {
  try {
    const translated = sp(label)
    return translated !== label ? translated : label
  } catch {
    return label
  }
}

function formatFieldValue(field: StrategyDetailField): string {
  const { value, format } = field
  if (typeof value === 'string') return value

  switch (format) {
    case 'price':
      return `$${value.toLocaleString()}`
    case 'percent':
      return `${value.toFixed(2)}%`
    case 'quantity':
      return String(value)
    default:
      return value.toLocaleString()
  }
}
