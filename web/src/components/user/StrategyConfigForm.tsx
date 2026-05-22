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

  function handleChange(key: string, value: unknown) {
    onChange({ ...values, [key]: value })
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
        {t('strategyParams')}
      </h3>
      {fields.map((field) => (
        <div key={field.key}>
          <label className="block text-sm font-medium mb-1">
            {field.label}
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
          ) : field.type === 'boolean' ? (
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={Boolean(values[field.key] ?? field.default)}
                onChange={(e) => handleChange(field.key, e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">{field.label}</span>
            </label>
          ) : (
            <input
              type={field.type === 'number' ? 'number' : 'text'}
              value={String(values[field.key] ?? field.default)}
              onChange={(e) => {
                const val = field.type === 'number' ? (e.target.value === '' ? '' : Number(e.target.value)) : e.target.value
                handleChange(field.key, val)
              }}
              required={field.required}
              min={field.min}
              max={field.max}
              step={field.step}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          )}
        </div>
      ))}
    </div>
  )
}
