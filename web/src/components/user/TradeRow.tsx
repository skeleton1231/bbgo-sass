'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { BBGoTrade } from '@/lib/bbgo/queries'

type SpotTag = 'open' | 'close' | 'add' | 'reduce'
type FuturesTag = 'openLong' | 'closeLong' | 'openShort' | 'closeShort'
  | 'addLong' | 'reduceLong' | 'addShort' | 'reduceShort'
  | 'flipLongToShort' | 'flipShortToLong'

const LONG_TAGS = new Set<string>(['openLong', 'closeLong', 'addLong', 'reduceLong', 'flipShortToLong'])
const SHORT_TAGS = new Set<string>(['openShort', 'closeShort', 'addShort', 'reduceShort', 'flipLongToShort'])

function futuresSideFromTag(tag: string | null | undefined): 'Long' | 'Short' | null {
  if (!tag) return null
  if (LONG_TAGS.has(tag)) return 'Long'
  if (SHORT_TAGS.has(tag)) return 'Short'
  return null
}

function positionActionToTag(action: string | null | undefined): SpotTag | FuturesTag | null {
  if (!action) return null
  const map: Record<string, SpotTag | FuturesTag> = {
    OPEN: 'open', ADD: 'add', REDUCE: 'reduce', CLOSE: 'close',
    OPEN_LONG: 'openLong', ADD_LONG: 'addLong', REDUCE_LONG: 'reduceLong', CLOSE_LONG: 'closeLong',
    OPEN_SHORT: 'openShort', ADD_SHORT: 'addShort', REDUCE_SHORT: 'reduceShort', CLOSE_SHORT: 'closeShort',
    FLIP_LONG_TO_SHORT: 'flipLongToShort', FLIP_SHORT_TO_LONG: 'flipShortToLong',
  }
  return map[action] ?? null
}

const SPOT_STYLES: Record<string, { border: string; text: string }> = {
  open:   { border: 'border-l-blue-400',    text: 'text-blue-400' },
  close:  { border: 'border-l-orange-400',   text: 'text-orange-400' },
  add:    { border: 'border-l-emerald-400',  text: 'text-emerald-400' },
  reduce: { border: 'border-l-amber-400',    text: 'text-amber-400' },
}

const FUTURES_STYLES: Record<string, { border: string; text: string }> = {
  openLong:        { border: 'border-l-blue-400',    text: 'text-blue-400' },
  addLong:         { border: 'border-l-blue-400',    text: 'text-blue-400' },
  closeLong:       { border: 'border-l-sky-400',     text: 'text-sky-400' },
  reduceLong:      { border: 'border-l-sky-400',     text: 'text-sky-400' },
  openShort:       { border: 'border-l-rose-400',    text: 'text-rose-400' },
  addShort:        { border: 'border-l-rose-400',    text: 'text-rose-400' },
  closeShort:      { border: 'border-l-orange-400',  text: 'text-orange-400' },
  reduceShort:     { border: 'border-l-orange-400',  text: 'text-orange-400' },
  flipLongToShort: { border: 'border-l-purple-400',  text: 'text-purple-400' },
  flipShortToLong: { border: 'border-l-purple-400',  text: 'text-purple-400' },
}

interface TradeRowProps {
  trade: BBGoTrade
  netPosition?: number
  isFutures?: boolean
}

export function TradeRow({ trade, netPosition = 0, isFutures }: TradeRowProps) {
  const t = useTranslations('Bots')
  const isBuy = trade.side === 'BUY'

  const tag = positionActionToTag(trade.positionAction ?? null)
  const futuresSide = isFutures ? (futuresSideFromTag(tag) ?? (isBuy ? 'Long' : 'Short')) : null
  const isFuturesTag = tag && (LONG_TAGS.has(tag) || SHORT_TAGS.has(tag))
  const styles = isFuturesTag ? FUTURES_STYLES[tag] : tag ? SPOT_STYLES[tag] : null
  const borderClass = styles?.border ?? (isBuy ? 'border-l-trade-up' : 'border-l-trade-down')

  const label = isFuturesTag
    ? t(`tradeTags.futures.${tag}` as `tradeTags.futures.${FuturesTag}`)
    : tag && tag in SPOT_STYLES
      ? t(`tradeTags.${tag}` as `tradeTags.${SpotTag}`)
      : null

  return (
    <div className={cn('flex items-center justify-between px-6 py-3 border-l-2', borderClass)}>
      <div className="flex items-center gap-3 min-w-0">
        <div className={cn(
          'flex items-center justify-center rounded text-xs font-bold',
          futuresSide ? 'h-6 px-2' : 'h-6 w-6',
          isBuy ? 'bg-trade-up/10 text-trade-up' : 'bg-trade-down/10 text-trade-down'
        )}>
          {futuresSide ?? (isBuy ? 'B' : 'S')}
        </div>
        <div className="flex flex-col gap-0.5 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium truncate">{trade.symbol}</span>
            {label && (
              <Badge variant="outline" className={cn('rounded-md text-[10px]', styles?.text)}>
                {label}
              </Badge>
            )}
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
