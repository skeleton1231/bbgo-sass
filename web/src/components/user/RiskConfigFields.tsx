'use client'

import { useTranslations } from 'next-intl'
import type { RiskConfig } from '@/lib/bbgo/strategies'

interface RiskConfigFieldsProps {
  value: RiskConfig
  onChange: (value: RiskConfig) => void
  symbol?: string
}

export function RiskConfigFields({ value, onChange, symbol }: RiskConfigFieldsProps) {
  const t = useTranslations('Risk')

  const update = (patch: Partial<RiskConfig>) => {
    onChange({ ...value, ...patch })
  }

  const numOrEmpty = (n: number | undefined) => (n === undefined || n === 0 ? '' : String(n))

  return (
    <div className="space-y-3 rounded-md border border-border p-3">
      <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
        {t('title')}
      </h4>
      <p className="text-xs text-muted-foreground">{t('overview')}</p>

      <div className="grid grid-cols-2 gap-3">
        <NumberField
          label={t('stopLossPrice')}
          hint={t('stopLossHint', { symbol: symbol ?? 'BTCUSDT' })}
          value={numOrEmpty(value.stopLossPrice)}
          step="0.0001"
          onChange={(v) => update({ stopLossPrice: v })}
        />
        <NumberField
          label={t('takeProfitPrice')}
          hint={t('takeProfitHint', { symbol: symbol ?? 'BTCUSDT' })}
          value={numOrEmpty(value.takeProfitPrice)}
          step="0.0001"
          onChange={(v) => update({ takeProfitPrice: v })}
        />
        <NumberField
          label={t('roiStopLoss')}
          hint={t('roiStopLossHint')}
          value={numOrEmpty(value.roiStopLoss)}
          step="0.001"
          onChange={(v) => update({ roiStopLoss: v })}
        />
        <NumberField
          label={t('roiTakeProfit')}
          hint={t('roiTakeProfitHint')}
          value={numOrEmpty(value.roiTakeProfit)}
          step="0.001"
          onChange={(v) => update({ roiTakeProfit: v })}
        />
        <NumberField
          label={t('trailingActivation')}
          hint={t('trailingActivationHint')}
          value={numOrEmpty(value.trailingActivation)}
          step="0.001"
          onChange={(v) => update({ trailingActivation: v })}
        />
        <NumberField
          label={t('trailingCallback')}
          hint={t('trailingCallbackHint')}
          value={numOrEmpty(value.trailingCallback)}
          step="0.001"
          onChange={(v) => update({ trailingCallback: v })}
        />
        <NumberField
          label={t('maxPositionQty')}
          hint={t('maxPositionQtyHint', { symbol: symbol ?? 'BTCUSDT' })}
          value={numOrEmpty(value.maxPositionQty)}
          step="0.0001"
          onChange={(v) => update({ maxPositionQty: v })}
        />
      </div>
    </div>
  )
}

interface NumberFieldProps {
  label: string
  hint?: string
  value: string
  step?: string
  onChange: (value: number | undefined) => void
}

function NumberField({ label, hint, value, step = '0.01', onChange }: NumberFieldProps) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">{label}</label>
      {hint && <p className="text-xs text-muted-foreground mb-1">{hint}</p>}
      <input
        type="number"
        inputMode="decimal"
        step={step}
        min={0}
        value={value}
        onChange={(e) => {
          const raw = e.target.value
          if (raw === '') {
            onChange(undefined)
            return
          }
          const parsed = Number(raw)
          onChange(Number.isFinite(parsed) ? parsed : undefined)
        }}
        className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
      />
    </div>
  )
}
