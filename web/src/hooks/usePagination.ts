'use client'

import { useCallback, useState } from 'react'

export interface UsePaginationOptions {
  initialPage?: number
  initialPageSize?: number
  initialTotal?: number
  pageSizeOptions?: number[]
}

export interface PaginationState {
  page: number
  pageSize: number
  total: number
  totalPages: number
  hasNext: boolean
  hasPrev: boolean
  /** Inclusive 0-based offset — pairs with Supabase `.range(from, to)` as `from`. */
  offset: number
  /** Inclusive 0-based end for Supabase `.range(from, to)`. */
  endIndex: number
  /** 1-based display range, e.g. `{ from: 21, to: 40 }` for page 2 / pageSize 20. */
  displayFrom: number
  displayTo: number
  setPage: (page: number) => void
  next: () => void
  prev: () => void
  setPageSize: (size: number) => void
  /** Update the row-count total. Typically called from a useEffect that watches
   *  the data query's `total` field. Page is clamped inline so we never render
   *  past the last available row. */
  setTotal: (total: number) => void
  pageSizeOptions: number[]
  reset: () => void
}

/**
 * Offset-based pagination hook shared by all list surfaces in the SaaS web app.
 *
 * Total is owned internally so callers can declare the pager *before* the data
 * query (avoiding chicken-and-egg on hook ordering) and push the server count
 * in via `setTotal` once the query resolves. The clamp runs inside `setTotal`
 * and `setPageSize` (not in an effect) so we don't trigger cascading renders.
 */
export function usePagination(opts: UsePaginationOptions = {}): PaginationState {
  const { initialPage = 1, initialPageSize = 20, initialTotal = 0, pageSizeOptions = [20, 50, 100, 200] } = opts

  const [page, setPageState] = useState(Math.max(1, initialPage))
  const [pageSize, setPageSizeState] = useState(initialPageSize)
  const [total, setTotalState] = useState(Math.max(0, initialTotal))

  const totalPages = total > 0 ? Math.max(1, Math.ceil(total / pageSize)) : 1

  const clampPage = useCallback(
    (nextTotal: number, nextPageSize: number, prevPage: number) => {
      const tp = nextTotal > 0 ? Math.max(1, Math.ceil(nextTotal / nextPageSize)) : 1
      return prevPage > tp ? tp : prevPage
    },
    []
  )

  const setPage = useCallback(
    (next: number) => {
      setPageState(Math.max(1, Math.min(next, totalPages)))
    },
    [totalPages]
  )

  const next = useCallback(() => setPage(page + 1), [page, setPage])
  const prev = useCallback(() => setPage(page - 1), [page, setPage])

  const setPageSize = useCallback((size: number) => {
    setPageSizeState(size)
    setPageState((prevPage) => Math.max(1, prevPage)) // reset to first page on size change
  }, [])

  const setTotal = useCallback(
    (nextTotal: number) => {
      const sanitized = Math.max(0, Math.floor(nextTotal))
      setTotalState(sanitized)
      setPageState((prevPage) => clampPage(sanitized, pageSize, prevPage))
    },
    [pageSize, clampPage]
  )

  const reset = useCallback(() => {
    setPageState(1)
  }, [])

  const offset = (page - 1) * pageSize
  const endIndex = offset + pageSize - 1
  const displayFrom = total === 0 ? 0 : offset + 1
  const displayTo = Math.min(offset + pageSize, total)

  return {
    page,
    pageSize,
    total,
    totalPages,
    hasNext: page < totalPages,
    hasPrev: page > 1,
    offset,
    endIndex,
    displayFrom,
    displayTo,
    setPage,
    next,
    prev,
    setPageSize,
    setTotal,
    pageSizeOptions,
    reset,
  }
}
