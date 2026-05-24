'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { useTranslations } from 'next-intl'
import {
  useUserStrategies,
  useStartUser,
  useStopUser,
  useBotSessions,
  useBotOpenOrders,
  useBotClosedOrders,
  useBotTrades,
  useBotSessionBalances,
  useBotStrategiesState,
  useBotPing,
  useContainerLogs,
  useBotPnL,
  type BBGoOrder,
  type BBGoTrade,
  type BBGoBalance,
} from '@/lib/bbgo/queries'
import { useMarketData } from '@/lib/bbgo/useWebSocket'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

export default function BotDetailPage() {
  const t = useTranslations('Bots')
  const router = useRouter()
  const params = useParams<{ id: string }>()
  const userId = params.id
  const [activeSession, setActiveSession] = useState<string>('')

  const { data: userContainer, isLoading, isError } = useUserStrategies(userId)
  const startUser = useStartUser()
  const stopUser = useStopUser()

  const { data: sessionsData } = useBotSessions(userId)
  const sessions = sessionsData?.sessions ?? []
  const firstSession = sessions[0]?.name ?? ''

  useEffect(() => {
    if (!activeSession && firstSession) {
      setActiveSession(firstSession)
    }
  }, [firstSession, activeSession])

  const isRunning = userContainer?.status === 'running'
  const { data: openOrdersData } = useBotOpenOrders(userId, activeSession)
  const { data: closedOrdersData } = useBotClosedOrders(userId)
  const { data: tradesData } = useBotTrades(userId)
  const { data: balancesData } = useBotSessionBalances(userId, activeSession)
  const { data: strategyStatesData } = useBotStrategiesState(userId)
  const { data: pingData } = useBotPing(userId)
  const { data: logsData } = useContainerLogs(userId, '100')
  const { data: pnlData } = useBotPnL(userId)
  const { connected: wsConnected } = useMarketData({ userId, enabled: isRunning })

  if (isLoading) {
    return <div className="text-muted-foreground">{t('loading')}</div>
  }

  if (isError) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6 text-center">
        <p className="text-sm text-destructive">{t('errorLoading')}</p>
      </div>
    )
  }

  const strategies = userContainer?.strategies ?? []
  const status = userContainer?.status ?? 'stopped'
  const openOrders = openOrdersData?.orders ?? []
  const closedOrders = closedOrdersData?.orders ?? []
  const trades = tradesData?.trades ?? []
  const balances = balancesData?.balances ?? {}
  const liveStrategies = strategyStatesData?.strategies ?? []
  const botReachable = isRunning && pingData?.status === 'ok'
  const nonZeroBalances = Object.entries(balances).filter(
    ([, b]) => parseFloat(b.available) > 0 || parseFloat(b.locked) > 0
  )

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <button onClick={() => router.back()} className="text-sm text-muted-foreground hover:text-foreground mb-2">
            &larr; {t('backToBots')}
          </button>
          <h1 className="text-2xl font-bold">{t('tradingDashboard')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('strategiesCount', { count: strategies.length })} · {t('containerName', { id: userId.slice(0, 8) })}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {isRunning && (
            <span className={cn(
              'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
              wsConnected ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
            )}>
              <span className={cn(
                'h-1.5 w-1.5 rounded-full',
                wsConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
              )} />
              {wsConnected ? t('live.connected') : t('live.disconnected')}
            </span>
          )}
          <span
            className={cn(
              'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
              status === 'running' && 'bg-green-100 text-green-700',
              status === 'stopped' && 'bg-gray-100 text-gray-700',
              status === 'error' && 'bg-red-100 text-red-700'
            )}
          >
            {t(`status.${status}`)}
          </span>
          {status === 'running' ? (
            <button
              onClick={() => stopUser.mutate(userId, { onError: (err) => toast.error(err.message) })}
              disabled={stopUser.isPending}
              className="rounded-md border px-4 py-2 text-sm hover:bg-muted disabled:opacity-50"
            >
              {t('stop')}
            </button>
          ) : (
            <button
              onClick={() => startUser.mutate(userId, { onError: (err) => toast.error(err.message) })}
              disabled={startUser.isPending}
              className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {t('start')}
            </button>
          )}
        </div>
      </div>

      {!isRunning && (
        <div className={cn(
          'rounded-lg border px-4 py-3 text-sm',
          status === 'error' ? 'bg-red-50 border-red-200 text-red-800' : 'bg-muted text-muted-foreground'
        )}>
          {status === 'error'
            ? t('errorBanner')
            : t('stoppedBanner')}
        </div>
      )}

      {/* Session tabs */}
      {sessions.length > 0 && (
        <div className="flex gap-2">
          {sessions.map((s) => (
            <button
              key={s.name}
              onClick={() => setActiveSession(s.name)}
              className={cn(
                'rounded-md px-3 py-1.5 text-sm font-medium',
                activeSession === s.name
                  ? 'bg-primary text-primary-foreground'
                  : 'border hover:bg-muted'
              )}
            >
              {s.exchangeName} ({s.name})
            </button>
          ))}
        </div>
      )}

      {/* PnL Dashboard */}
      {botReachable && pnlData && pnlData.totalTrades > 0 && (
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-4">
            <div className="rounded-lg border bg-card p-4">
              <p className="text-xs text-muted-foreground">{t('pnl.realized')}</p>
              <p className={cn(
                'text-xl font-bold',
                pnlData.totalRealizedPnl > 0 ? 'text-green-600' : pnlData.totalRealizedPnl < 0 ? 'text-red-600' : ''
              )}>
                {pnlData.totalRealizedPnl >= 0 ? '+' : ''}{pnlData.totalRealizedPnl.toFixed(4)} USDT
              </p>
            </div>
            <div className="rounded-lg border bg-card p-4">
              <p className="text-xs text-muted-foreground">{t('pnl.totalFees')}</p>
              <p className="text-xl font-bold text-muted-foreground">
                -{pnlData.totalFees.toFixed(4)} USDT
              </p>
            </div>
            <div className="rounded-lg border bg-card p-4">
              <p className="text-xs text-muted-foreground">{t('pnl.winRate')}</p>
              <p className="text-xl font-bold">
                {pnlData.winRate.toFixed(1)}%
                <span className="text-xs text-muted-foreground ml-2">
                  ({pnlData.winningTrades}W / {pnlData.losingTrades}L)
                </span>
              </p>
            </div>
            <div className="rounded-lg border bg-card p-4">
              <p className="text-xs text-muted-foreground">{t('pnl.totalTrades')}</p>
              <p className="text-xl font-bold">{pnlData.totalTrades}</p>
            </div>
          </div>

          {pnlData.symbols.length > 0 && (
            <div className="rounded-lg border bg-card">
              <div className="p-4 border-b">
                <h2 className="font-semibold">{t('pnl.bySymbol')}</h2>
              </div>
              <div className="divide-y">
                {pnlData.symbols.map((s) => (
                  <div key={s.symbol} className="flex items-center justify-between px-4 py-3 text-sm">
                    <div className="flex items-center gap-3 min-w-[140px]">
                      <span className="font-medium">{s.symbol}</span>
                      <span className="text-xs text-muted-foreground">{s.tradeCount} trades</span>
                    </div>
                    <div className="flex items-center gap-6">
                      {s.openPosition > 0 && (
                        <span className="text-xs text-muted-foreground">
                          pos: {s.openPosition.toFixed(6)} @ ~{s.openPositionCost > 0 ? (s.openPositionCost / s.openPosition).toFixed(2) : '-'}
                        </span>
                      )}
                      <span className="text-xs text-muted-foreground w-20 text-right">
                        avg buy {s.avgBuyPrice > 0 ? s.avgBuyPrice.toFixed(2) : '-'}
                      </span>
                      <span className={cn(
                        'font-medium w-32 text-right',
                        s.realizedPnl > 0 ? 'text-green-600' : s.realizedPnl < 0 ? 'text-red-600' : ''
                      )}>
                        {s.realizedPnl >= 0 ? '+' : ''}{s.realizedPnl.toFixed(4)}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        {/* Balances */}
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">{t('balances')}</h2>
          </div>
          {nonZeroBalances.length > 0 ? (
            <div className="divide-y max-h-80 overflow-y-auto">
              {nonZeroBalances.map(([currency, b]: [string, BBGoBalance]) => (
                <div key={currency} className="flex items-center justify-between px-4 py-2 text-sm">
                  <span className="font-medium">{currency}</span>
                  <div className="text-right">
                    <span>{b.available}</span>
                    {parseFloat(b.locked) > 0 && (
                      <span className="text-muted-foreground ml-1">({t('locked', { amount: b.locked })})</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="p-4 text-sm text-muted-foreground">
              {isRunning ? t('noBalances') : t('startToSeeData')}
            </div>
          )}
        </div>

        {/* Open Orders */}
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">{t('openOrders')} ({openOrders.length})</h2>
          </div>
          {openOrders.length > 0 ? (
            <div className="divide-y max-h-80 overflow-y-auto">
              {openOrders.map((order: BBGoOrder) => (
                <div key={order.orderID} className="flex items-center justify-between px-4 py-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className={cn(
                      'text-xs font-medium rounded px-1.5 py-0.5',
                      order.side === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                    )}>
                      {order.side}
                    </span>
                    <span>{order.symbol}</span>
                    <span className="text-xs text-muted-foreground">{order.orderType}</span>
                  </div>
                  <div className="text-muted-foreground">
                    {order.price} x {order.quantity}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="p-4 text-sm text-muted-foreground">
              {isRunning ? t('noOpenOrders') : t('startToSeeData')}
            </div>
          )}
        </div>
      </div>

      {/* Closed Orders */}
      <div className="rounded-lg border bg-card">
        <div className="p-4 border-b">
          <h2 className="font-semibold">{t('closedOrders')} ({closedOrders.length})</h2>
        </div>
        {closedOrders.length > 0 ? (
          <div className="divide-y max-h-80 overflow-y-auto">
            {closedOrders.map((order: BBGoOrder) => (
              <div key={order.orderID} className="flex items-center justify-between px-4 py-2 text-sm">
                <div className="flex items-center gap-2">
                  <span className={cn(
                    'text-xs font-medium rounded px-1.5 py-0.5',
                    order.side === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                  )}>
                    {order.side}
                  </span>
                  <span className="font-medium">{order.symbol}</span>
                  <span className="text-xs text-muted-foreground">{order.orderType}</span>
                </div>
                <div className="flex items-center gap-4 text-muted-foreground">
                  <span>{order.price} x {order.executedQuantity || order.quantity}</span>
                  {order.status && (
                    <span className={cn(
                      'text-xs rounded px-1.5 py-0.5',
                      order.status === 'Filled' ? 'bg-green-50 text-green-600' :
                      order.status === 'Canceled' ? 'bg-gray-50 text-gray-500' :
                      'bg-yellow-50 text-yellow-600'
                    )}>
                      {order.status}
                    </span>
                  )}
                  {order.creationTime && (
                    <span className="text-xs">{new Date(order.creationTime).toLocaleString()}</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="p-4 text-sm text-muted-foreground">
            {isRunning ? t('noClosedOrders') : t('startToSeeData')}
          </div>
        )}
      </div>

      {/* Trades */}
      <div className="rounded-lg border bg-card">
        <div className="p-4 border-b">
          <h2 className="font-semibold">{t('recentTrades')}</h2>
        </div>
        {trades.length > 0 ? (
          <div className="divide-y max-h-80 overflow-y-auto">
            {trades.map((trade: BBGoTrade) => (
              <div key={trade.id} className="flex items-center justify-between px-4 py-2 text-sm">
                <div className="flex items-center gap-3">
                  <span className={cn(
                    'text-xs font-medium rounded px-1.5 py-0.5',
                    trade.side === 'BUY' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                  )}>
                    {trade.side}
                  </span>
                  <span className="font-medium">{trade.symbol}</span>
                  <span className="text-xs text-muted-foreground">{trade.exchange}</span>
                  {trade.isMaker && (
                    <span className="text-xs text-muted-foreground border rounded px-1">{t('maker')}</span>
                  )}
                </div>
                <div className="flex items-center gap-4 text-muted-foreground">
                  <span>{trade.price} x {trade.quantity}</span>
                  {trade.quoteQuantity && parseFloat(trade.quoteQuantity) > 0 && (
                    <span className="text-xs">${parseFloat(trade.quoteQuantity).toFixed(2)}</span>
                  )}
                  <span className="text-xs">{trade.fee} {trade.feeCurrency}</span>
                  {trade.tradedAt && (
                    <span className="text-xs">
                      {new Date(trade.tradedAt).toLocaleString()}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="p-4 text-sm text-muted-foreground">
            {isRunning ? t('noTrades') : t('startToSeeData')}
          </div>
        )}
      </div>

      {/* Strategies */}
      {strategies.length > 0 && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b flex items-center justify-between">
            <h2 className="font-semibold">{t('strategies')} ({strategies.length})</h2>
            {liveStrategies.length > 0 && (
              <span className="text-xs text-green-600">{t('activeCount', { count: liveStrategies.length })}</span>
            )}
          </div>
          <div className="divide-y">
            {strategies.map((s) => {
              const liveState = liveStrategies.find((ls) => ls.strategy === s.strategy)
              return (
                <div key={s.id} className="px-4 py-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium">{s.name || s.strategy}</p>
                      <p className="text-xs text-muted-foreground">
                        {s.exchange}{s.crossExchange ? ' (cross-exchange)' : ''} · {s.strategy} · {t(`mode.${s.mode}`)}
                      </p>
                    </div>
                    {isRunning && (
                      <span className={cn(
                        'text-xs font-medium rounded-full px-2 py-0.5',
                        liveState ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                      )}>
                        {liveState ? t('strategyStatus.running') : t('strategyStatus.idle')}
                      </span>
                    )}
                  </div>
                  {liveState && Object.keys(liveState).length > 1 && (
                    <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
                      {Object.entries(liveState)
                        .filter(([k]) => k !== 'strategy')
                        .slice(0, 6)
                        .map(([key, val]) => (
                          <span key={key} className="text-xs text-muted-foreground">
                            {key}: <span className="text-foreground">{String(val)}</span>
                          </span>
                        ))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Container Logs */}
      {isRunning && logsData?.logs && (
        <div className="rounded-lg border bg-card">
          <div className="p-4 border-b">
            <h2 className="font-semibold">{t('containerLogs')}</h2>
          </div>
          <pre className="whitespace-pre-wrap text-xs text-muted-foreground max-h-[300px] overflow-y-auto p-4">
            {logsData.logs.replace(/\x1b\[[0-9;]*m/g, '')}
          </pre>
        </div>
      )}
    </div>
  )
}
