'use client'

import type { PnLReport } from '@/lib/bbgo/queries'

interface PnlSummaryProps {
  report: PnLReport
}

export function PnlSummary({ report }: PnlSummaryProps) {
  const isPositive = report.totalRealizedPnl >= 0

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div className="space-y-1">
        <p className="text-xs text-muted-foreground">Realized P&L</p>
        <p className={isPositive ? 'text-green-600 font-bold' : 'text-red-600 font-bold'}>
          {isPositive ? '+' : ''}${report.totalRealizedPnl.toFixed(2)}
        </p>
      </div>
      <div className="space-y-1">
        <p className="text-xs text-muted-foreground">Total Fees</p>
        <p className="text-sm font-medium">${report.totalFees.toFixed(2)}</p>
      </div>
      <div className="space-y-1">
        <p className="text-xs text-muted-foreground">Win Rate</p>
        <p className="text-sm font-medium">
          {report.totalTrades > 0
            ? `${((report.winningTrades / report.totalTrades) * 100).toFixed(1)}%`
            : '--'}
        </p>
        <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
          <div
            className="h-full bg-green-500 rounded-full transition-all"
            style={{
              width: report.totalTrades > 0
                ? `${(report.winningTrades / report.totalTrades) * 100}%`
                : '0%',
            }}
          />
        </div>
      </div>
      <div className="space-y-1">
        <p className="text-xs text-muted-foreground">Trades</p>
        <p className="text-sm font-medium">
          {report.totalTrades}
          <span className="text-xs text-muted-foreground ml-1">
            ({report.winningTrades}W / {report.losingTrades}L)
          </span>
        </p>
      </div>
    </div>
  )
}
