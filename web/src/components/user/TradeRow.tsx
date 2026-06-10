'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { BBGoTrade } from '@/lib/bbgo/queries'

type PositionAction = 'open_long' | 'open_short' | 'close_long' | 'close_short'
type LocalTag = 'open' | 'close' | 'add' | 'reduce' | 'trade'

interface TradeRowProps {
  trade: BBGoTrade
  tag?: string | null
  netPosition?: number
}

const PA_STYLES: Record<string, { border: string; text: string }> = {
  open_long:   { border: 'border-blue-400',   text: 'text-blue-400' },
  open_short:  { border: 'border-rose-400',   text: 'text-rose-400' },
  close_long:  { border: 'border-sky-400',    text: 'text-sky-400' },
  close_short: { border: 'border-orange-400', text: 'text-orange-400' },
}

function getBorderClass(serverPa: PositionAction | undefined, tag: string | null | undefined, isBuy: boolean): string {
  if (serverPa) return PA_STYLES[serverPa]?.border ?? (isBuy ? 'border-l-trade-up' : 'border-l-trade-down')
  if (tag === 'open') return 'border-l-blue-400'
  if (tag === 'close') return 'border-l-orange-400'
  return isBuy ? 'border-l-trade-up' : 'border-l-trade-down'
}

export function TradeRow({ trade, tag, netPosition = 0 }: TradeRowProps) {
  const t = useTranslations('Bots')
  const isBuy = trade.side === 'BUY'
  const serverPa = trade.serverPositionAction as PositionAction | undefined
  const paStyle = serverPa ? PA_STYLES[serverPa] : null
  const borderClass = getBorderClass(serverPa, tag, isBuy)

  return (
    <div className={cn('flex items-center justify-between px-6 py-3 border-l-2', borderClass)}>
      <div className="flex items-center gap-3 min-w-0">
        <div className={cn(
          'flex h-6 w-6 items-center justify-center rounded text-xs font-bold',
          isBuy ? 'bg-trade-up/10 text-trade-up' : 'bg-trade-down/10 text-trade-down'
        )}>
          {isBuy ? 'B' : 'S'}
        </div>
        <div className="flex flex-col gap-0.5 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium truncate">{trade.symbol}</span>
            {serverPa && (
              <Badge variant="outline" className={cn('rounded-md text-[10px]', paStyle?.border, paStyle?.text)}>
                {t(`positionAction.${serverPa}`)}
              </Badge>
            )}
            {!serverPa && tag === 'open' && <Badge variant="outline" className="rounded-md text-[10px] border-blue-400 text-blue-400">{t('tradeTags.open')}</Badge>}
            {!serverPa && tag === 'close' && <Badge variant="outline" className="rounded-md text-[10px] border-orange-400 text-orange-400">{t('tradeTags.close')}</Badge>}
            {!serverPa && tag === 'add' && <Badge variant="outline" className="rounded-md text-[10px] border-emerald-400 text-emerald-400">{t('tradeTags.add')}</Badge>}
            {!serverPa && tag === 'reduce' && <Badge variant="outline" className="rounded-md text-[10px] border-amber-400 text-amber-400">{t('tradeTags.reduce')}</Badge>}
            {trade.isMaker && <Badge variant="outline" className="rounded-md text-[10px]">{t('tradeTags.maker')}</Badge>}
            <span className="text-[10px] text-muted-foreground tabular-nums">{t('tradeTags.net', { position: netPosition.toFixed(6) })}</span>
          </div>
          <span className="text-xs text-muted-foreground">{trade.exchange}</span>
        </div>
      </div>
      <div className="text-right space-y-0.5 shrink-0">
        <p className="text-sm font-mono">{trade.price} × {trade.quantity}</p>
        <div className="flex items-center justify-end gap-3 text-xs text-muted-foreground">
          {trade.quoteQuantity && parseFloat(trade.quoteQuantity) > 0 && (
            <span>${parseFloat(trade.quoteQuantity).toFixed(2)}</span>
          )}
          <span>{trade.fee} {trade.feeCurrency}</span>
          {trade.tradedAt && <span className="tabular-nums">{new Date(trade.tradedAt).toLocaleString()}</span>}
        </div>
      </div>
    </div>
  )
}
