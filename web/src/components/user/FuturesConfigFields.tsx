'use client'

import { useTranslations } from 'next-intl'
import type { FuturesConfig } from '@/lib/bbgo/strategies'

interface FuturesConfigFieldsProps {
  value: FuturesConfig
  onChange: (value: FuturesConfig) => void
  symbol?: string
}

export function FuturesConfigFields({ value, onChange, symbol }: FuturesConfigFieldsProps) {
  const t = useTranslations('Futures')

  return (
    <div className="space-y-3 rounded-md border border-border p-3">
      <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
        {t('title')}
      </h4>

      <div>
        <label className="block text-sm font-medium mb-1">
          {t('leverage')}
        </label>
        <p className="text-xs text-muted-foreground mb-1">
          {t('leverageHint', { symbol: symbol ?? 'BTCUSDT' })}
        </p>
        <div className="flex items-center gap-3">
          <input
            type="range"
            min={1}
            max={125}
            value={value.leverage ?? 1}
            onChange={(e) => onChange({ ...value, leverage: Number(e.target.value) })}
            className="flex-1 h-2 rounded-lg appearance-none cursor-pointer accent-primary bg-border"
          />
          <span className="w-12 text-center text-sm font-mono font-semibold">
            {value.leverage ?? 1}x
          </span>
        </div>
        <div className="flex gap-1 mt-1.5">
          {[1, 2, 3, 5, 10, 20].map((lev) => (
            <button
              key={lev}
              type="button"
              onClick={() => onChange({ ...value, leverage: lev })}
              className={`px-2 py-0.5 text-xs rounded border transition-colors ${
                (value.leverage ?? 1) === lev
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border hover:border-primary/50'
              }`}
            >
              {lev}x
            </button>
          ))}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">
          {t('marginTypeLabel')}
        </label>
        <p className="text-xs text-muted-foreground mb-1">
          {t('marginTypeHint')}
        </p>
        <div className="flex gap-3">
          {(['cross', 'isolated'] as const).map((mt) => (
            <label
              key={mt}
              className={`flex items-center gap-2 text-sm cursor-pointer rounded-md border px-3 py-2 transition-colors ${
                (value.marginType ?? 'cross') === mt
                  ? 'border-primary bg-primary/5'
                  : 'border-border hover:border-primary/30'
              }`}
            >
              <input
                type="radio"
                name="marginType"
                value={mt}
                checked={(value.marginType ?? 'cross') === mt}
                onChange={() => onChange({ ...value, marginType: mt })}
                className="h-3.5 w-3.5 accent-primary"
              />
              <span>{t(`marginType.${mt}`)}</span>
            </label>
          ))}
        </div>
      </div>
    </div>
  )
}
