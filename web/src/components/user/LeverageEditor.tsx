'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useUpdateStrategy } from '@/lib/bbgo/queries'
import { useUserId } from '@/components/providers/user-id'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'

interface LeverageEditorProps {
  instanceId: string
  strategy: string
  symbol: string
  currentLeverage?: number
  currentMarginType?: 'cross' | 'isolated'
  requiresFutures?: boolean
}

export function LeverageEditor({
  instanceId,
  strategy,
  symbol,
  currentLeverage,
  currentMarginType,
  requiresFutures,
}: LeverageEditorProps) {
  const t = useTranslations('Bots')
  const userId = useUserId()
  const updateStrategy = useUpdateStrategy()
  const [editing, setEditing] = useState(false)
  const [leverage, setLeverage] = useState<number>(currentLeverage ?? 1)
  const [marginType, setMarginType] = useState<'cross' | 'isolated'>(currentMarginType ?? 'cross')

  if (!requiresFutures) return null

  const onSave = () => {
    // Number.isFinite rejects NaN (from non-numeric input) which would otherwise
    // pass the range check (NaN < 1 is false) and silently serialize to null.
    if (!Number.isFinite(leverage) || leverage < 1 || leverage > 125) {
      toast.error(t('leverageRangeError'))
      return
    }
    updateStrategy.mutate(
      { userId, strategyId: instanceId, futuresConfig: { leverage, marginType } },
      {
        onSuccess: (data) => {
          toast.success(t('leverageUpdated', { leverage: data.futuresConfig?.leverage ?? leverage }))
          setEditing(false)
        },
        onError: (err: Error) => toast.error(err.message),
      },
    )
  }

  if (!editing) {
    return (
      <div className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground">{t('leverageLabel')}:</span>
        <span className="font-mono font-semibold">{currentLeverage ?? '-'}x</span>
        <span className="text-muted-foreground">·</span>
        <span className="font-mono">{currentMarginType ?? '-'}</span>
        <Button variant="ghost" size="sm" onClick={() => setEditing(true)}>
          {t('edit')}
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-2 rounded-md border border-border p-3">
      <div className="text-sm font-medium">{t('editLeverageTitle', { strategy, symbol })}</div>
      <div className="flex items-end gap-3">
        <div className="flex-1">
          <Label className="text-xs">{t('leverageLabel')}</Label>
          <Input
            type="number"
            min={1}
            max={125}
            value={leverage}
            onChange={(e) => setLeverage(Number(e.target.value))}
            className="mt-1"
          />
        </div>
        <div>
          <Label className="text-xs">{t('marginTypeLabel')}</Label>
          <div className="mt-1 flex gap-1">
            {(['cross', 'isolated'] as const).map((mt) => (
              <button
                key={mt}
                type="button"
                onClick={() => setMarginType(mt)}
                className={cn(
                  'px-2 py-1 text-xs rounded border',
                  marginType === mt ? 'border-primary bg-primary/10' : 'border-border',
                )}
              >
                {mt}
              </button>
            ))}
          </div>
        </div>
      </div>
      <p className="text-xs text-yellow-600 bg-yellow-50 dark:bg-yellow-950/30 px-2 py-1 rounded">
        {t('leverageRestartHint')}
      </p>
      <div className="flex justify-end gap-2">
        <Button variant="ghost" size="sm" onClick={() => setEditing(false)} disabled={updateStrategy.isPending}>
          {t('cancel')}
        </Button>
        <Button size="sm" onClick={onSave} disabled={updateStrategy.isPending}>
          {updateStrategy.isPending ? t('saving') : t('save')}
        </Button>
      </div>
    </div>
  )
}
