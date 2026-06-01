'use client'

import { useTranslations } from 'next-intl'
import { type FieldDef } from '@/lib/bbgo/strategies'

interface StrategyConfigFormProps {
  fields: FieldDef[]
  values: Record<string, unknown>
  onChange: (values: Record<string, unknown>) => void
}

export function StrategyConfigForm({ fields, values, onChange }: StrategyConfigFormProps) {
  const t = useTranslations('Bots')
  const ft = useTranslations('StrategyFields')

  function handleChange(key: string, value: unknown) {
    onChange({ ...values, [key]: value })
  }

  function formatNumberValue(val: unknown, fallback: unknown): string {
    const raw = val ?? fallback
    if (raw === '' || raw === undefined) return ''
    if (typeof raw === 'string') return raw
    const num = Number(raw)
    if (!Number.isFinite(num)) return String(raw)
    const s = num.toPrecision(10)
    return parseFloat(s).toString()
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
        {t('strategyParams')}
      </h3>
      {fields.map((field) => (
        <div key={field.key}>
          {field.type === 'boolean' ? (
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={Boolean(values[field.key] ?? field.default)}
                onChange={(e) => handleChange(field.key, e.target.checked)}
                className="h-4 w-4 rounded border-input accent-primary"
              />
              <span className="text-sm font-medium">
                {ft(field.key as Parameters<typeof ft>[0])}
                {field.required && <span className="text-destructive ml-0.5">*</span>}
              </span>
            </label>
          ) : (
            <>
              <label className="block text-sm font-medium mb-1">
                {ft(field.key as Parameters<typeof ft>[0])}
                {field.required && <span className="text-destructive ml-0.5">*</span>}
              </label>

              {field.description && (
                <p className="text-xs text-muted-foreground mb-1">{field.description}</p>
              )}

              {field.type === 'select' ? (
                <select
                  value={String(values[field.key] ?? field.default)}
                  onChange={(e) => handleChange(field.key, e.target.value)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                >
                  {field.options?.map((opt) => (
                    <option key={opt} value={opt}>{opt}</option>
                  ))}
                </select>
              ) : (
            <input
              type="text"
              inputMode="decimal"
              value={field.type === 'number' ? formatNumberValue(values[field.key], field.default) : String(values[field.key] ?? field.default ?? '')}
              onChange={(e) => {
                handleChange(field.key, e.target.value)
              }}
              onBlur={() => {
                if (field.type === 'number') {
                  const raw = values[field.key]
                  if (raw === '' || raw === undefined) return
                  const num = Number(raw)
                  if (Number.isFinite(num)) {
                    handleChange(field.key, num)
                  }
                }
              }}
              required={field.required}
              min={field.min}
              max={field.max}
              step={field.step}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
              )}
            </>
          )}
        </div>
      ))}
    </div>
  )
}
