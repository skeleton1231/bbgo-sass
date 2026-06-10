'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { BBGoOrder } from '@/lib/bbgo/queries'

interface OrderRowProps {
  order: BBGoOrder
  showStatus?: boolean
  showTime?: boolean
}

const TRANSLATED_STATUSES = ['New', 'Filled', 'PartiallyFilled', 'Canceled', 'Rejected'] as const

export function OrderRow({ order, showStatus, showTime }: OrderRowProps) {
  const t = useTranslations('Bots')
  const statusLabel = (TRANSLATED_STATUSES as readonly string[]).includes(order.status)
    ? t(`orderStatus.${order.status}` as typeof TRANSLATED_STATUSES[number])
    : order.status
  const executed = parseFloat(order.executedQuantity || '0')
  const total = parseFloat(order.quantity)
  const fillPct = total > 0 ? Math.round((executed / total) * 100) : 0
  const isBuy = order.side === 'BUY'
  const sideColor = isBuy ? 'text-trade-up' : 'text-trade-down'
  const sideBg = isBuy ? 'bg-trade-up/10' : 'bg-trade-down/10'
  const borderClass = isBuy ? 'border-l-trade-up' : 'border-l-trade-down'

  return (
    <div className={cn(
      'flex items-center justify-between px-6 py-3 border-l-2',
      borderClass
    )}>
      <div className="flex items-center gap-3 min-w-0">
        <div className={cn('flex h-6 w-6 items-center justify-center rounded text-xs font-bold', sideBg, sideColor)}>
          {isBuy ? 'B' : 'S'}
        </div>
        <div className="flex flex-col gap-0.5 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium truncate">{order.symbol}</span>
            <Badge variant="secondary" className="rounded-md text-[10px] shrink-0">{order.orderType}</Badge>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-mono">{order.price}</span>
            <span>×</span>
            <span className="font-mono">{executed > 0 ? order.executedQuantity : order.quantity}</span>
            {executed > 0 && executed < total && (
              <span className={cn('text-[10px] px-1 rounded', sideBg, sideColor)}>
                {t('filled', { pct: fillPct })}
              </span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3 shrink-0">
        {showStatus && order.status && (
          <Badge variant="outline" className={cn(
            'rounded-full text-[10px]',
            order.status === 'Filled' && 'border-trade-up/30 text-trade-up',
            order.status === 'Canceled' && 'border-border text-muted-foreground',
            order.status === 'New' && 'border-blue-500/30 text-blue-500',
            order.status === 'PartiallyFilled' && 'border-yellow-500/30 text-yellow-600',
          )}>
            {statusLabel}
          </Badge>
        )}
        {showTime && order.creationTime && (
          <span className="text-xs text-muted-foreground tabular-nums">{new Date(order.creationTime).toLocaleString()}</span>
        )}
      </div>
    </div>
  )
}
