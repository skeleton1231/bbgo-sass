'use client'

import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslations } from 'next-intl'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'

interface PaginationControlsProps {
  page: number
  pageSize: number
  total: number
  totalPages: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
  pageSizeOptions?: number[]
  loading?: boolean
  className?: string
}

/**
 * Compact Prev/Next pager + page-size selector. Pairs with `usePagination`
 * for state, but accepts plain props so any caller can use it.
 */
export function PaginationControls({
  page,
  pageSize,
  total,
  totalPages,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [20, 50, 100, 200],
  loading = false,
  className,
}: PaginationControlsProps) {
  const t = useTranslations('Common.pagination')

  if (total === 0) return null

  const displayFrom = (page - 1) * pageSize + 1
  const displayTo = Math.min(page * pageSize, total)

  return (
    <div className={cn(
      'flex items-center justify-between gap-2 px-4 py-2 text-xs text-muted-foreground border-t',
      className
    )}>
      <div className="flex items-center gap-2">
        <span className="tabular-nums">
          {t('range', { from: displayFrom, to: displayTo, total })}
        </span>
        <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(parseInt(v, 10))} disabled={loading}>
          <SelectTrigger size="sm" className="h-7 w-[80px] text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {pageSizeOptions.map((s) => (
              <SelectItem key={s} value={String(s)}>{s} / {t('page')}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="icon-xs"
          disabled={page <= 1 || loading}
          onClick={() => onPageChange(page - 1)}
          aria-label={t('prev')}
        >
          <ChevronLeft />
        </Button>
        <span className="px-2 tabular-nums">
          {loading ? '—' : `${page} / ${totalPages}`}
        </span>
        <Button
          variant="outline"
          size="icon-xs"
          disabled={page >= totalPages || loading}
          onClick={() => onPageChange(page + 1)}
          aria-label={t('next')}
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  )
}
