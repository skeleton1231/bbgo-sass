'use client'

import { useTranslations } from 'next-intl'
import { type FieldDef } from '@/lib/bbgo/strategies'

interface StrategyConfigFormProps {
  fields: FieldDef[]
  values: Record<string, unknown>
  onChange: (values: Record<string, unknown>) => void
}

function FieldInput({ field, value, onChange }: {
  field: FieldDef
  value: unknown
  onChange: (key: string, value: unknown) => void
}) {
  const ft = useTranslations('StrategyFields')

  function formatNumberValue(val: unknown, fallback: unknown): string {
    const raw = val ?? fallback
    if (raw === '' || raw === undefined) return ''
    if (typeof raw === 'string') return raw
    const num = Number(raw)
    if (!Number.isFinite(num)) return String(raw)
    return parseFloat(num.toPrecision(10)).toString()
  }

  const label = ft(field.key as Parameters<typeof ft>[0])

  if (field.type === 'boolean') {
    return (
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={Boolean(value ?? field.default)}
          onChange={(e) => onChange(field.key, e.target.checked)}
          className="h-4 w-4 rounded border-input accent-primary"
        />
        <span className="text-sm font-medium">
          {label}
          {field.required && <span className="text-destructive ml-0.5">*</span>}
        </span>
      </label>
    )
  }

  return (
    <>
      <label className="block text-sm font-medium mb-1">
        {label}
        {field.required && <span className="text-destructive ml-0.5">*</span>}
      </label>
      {field.description && (
        <p className="text-xs text-muted-foreground mb-1">{field.description}</p>
      )}
      {field.type === 'select' ? (
        <select
          value={String(value ?? field.default)}
          onChange={(e) => onChange(field.key, e.target.value)}
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
          value={field.type === 'number' ? formatNumberValue(value, field.default) : String(value ?? field.default ?? '')}
          onChange={(e) => onChange(field.key, e.target.value)}
          onBlur={() => {
            if (field.type === 'number') {
              if (value === '' || value === undefined) return
              const num = Number(value)
              if (Number.isFinite(num)) onChange(field.key, num)
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
  )
}

export function StrategyConfigForm({ fields, values, onChange }: StrategyConfigFormProps) {
  const t = useTranslations('Bots')
  const ft = useTranslations('StrategyFields')

  function handleChange(key: string, value: unknown) {
    onChange({ ...values, [key]: value })
  }

  const ungrouped: FieldDef[] = []
  const groups = new Map<string, FieldDef[]>()

  for (const field of fields) {
    if (field.group) {
      const existing = groups.get(field.group)
      if (existing) existing.push(field)
      else groups.set(field.group, [field])
    } else {
      ungrouped.push(field)
    }
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
        {t('strategyParams')}
      </h3>

      {ungrouped.map((field) => (
        <div key={field.key}>
          <FieldInput field={field} value={values[field.key]} onChange={handleChange} />
        </div>
      ))}

      {[...groups.entries()].map(([groupName, groupFields]) => (
        <div key={groupName} className="space-y-3 rounded-md border border-border p-3">
          <h4 className="text-sm font-semibold text-muted-foreground">
            {ft(groupName as Parameters<typeof ft>[0])}
          </h4>
          {groupFields.map((field) => (
            <div key={field.key}>
              <FieldInput field={field} value={values[field.key]} onChange={handleChange} />
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}
