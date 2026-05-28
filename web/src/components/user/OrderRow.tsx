import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { ArrowDownRight, ArrowUpRight } from 'lucide-react'
import type { BBGoOrder } from '@/lib/bbgo/queries'

interface OrderRowProps {
  order: BBGoOrder
  showStatus?: boolean
  showTime?: boolean
}

export function OrderRow({ order, showStatus, showTime }: OrderRowProps) {
  return (
    <div className="flex items-center justify-between px-6 py-3">
      <div className="flex items-center gap-3">
        <div className={cn(
          'flex h-7 w-7 items-center justify-center rounded-full',
          order.side === 'BUY' ? 'bg-trade-up' : 'bg-trade-down'
        )}>
          {order.side === 'BUY'
            ? <ArrowDownRight className="h-3.5 w-3.5 text-trade-up" />
            : <ArrowUpRight className="h-3.5 w-3.5 text-trade-down" />}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{order.symbol}</span>
          <Badge variant="secondary" className="rounded-md text-[10px]">{order.orderType}</Badge>
        </div>
      </div>
      <div className="flex items-center gap-4">
        <span className="text-sm text-muted-foreground font-mono">
          {order.price} × {order.executedQuantity || order.quantity}
        </span>
        {showStatus && order.status && (
          <Badge variant="outline" className={cn(
            'rounded-full text-[10px]',
            order.status === 'Filled' && 'border-trade-up/30 text-trade-up',
            order.status === 'Canceled' && 'border-border text-muted-foreground'
          )}>
            {order.status}
          </Badge>
        )}
        {showTime && order.creationTime && (
          <span className="text-xs text-muted-foreground">{new Date(order.creationTime).toLocaleString()}</span>
        )}
      </div>
    </div>
  )
}
