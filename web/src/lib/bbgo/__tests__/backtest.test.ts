import { describe, it, expect } from 'vitest'
import type { BacktestJob } from '../manager'

describe('BacktestJob status types', () => {
  it('accepts all valid status values', () => {
    const statuses: BacktestJob['status'][] = [
      'pending',
      'downloading',
      'running',
      'completed',
      'failed',
    ]
    expect(statuses).toHaveLength(5)
  })
})

describe('BacktestJob type shape', () => {
  it('has required fields for a completed job', () => {
    const job: BacktestJob = {
      id: 'bt-123',
      user_id: 'user-1',
      strategy: 'grid2',
      config: { symbol: 'BTCUSDT', gridNumber: 10 },
      exchange: 'binance',
      symbol: 'BTCUSDT',
      start_time: '2024-01-01',
      end_time: '2024-03-01',
      status: 'completed',
      output: 'profit=1234.5',
      created_at: '2024-06-01T00:00:00Z',
      completed_at: '2024-06-01T00:05:00Z',
      need_sync: true,
    }
    expect(job.id).toBe('bt-123')
    expect(job.status).toBe('completed')
    expect(job.output).toBeDefined()
  })

  it('has optional fields for a pending job', () => {
    const job: BacktestJob = {
      id: 'bt-456',
      user_id: 'user-1',
      strategy: 'grid2',
      config: {},
      exchange: 'binance',
      symbol: 'BTCUSDT',
      start_time: '2024-01-01',
      end_time: '2024-03-01',
      status: 'pending',
      created_at: '2024-06-01T00:00:00Z',
      need_sync: true,
    }
    expect(job.progress).toBeUndefined()
    expect(job.output).toBeUndefined()
    expect(job.error).toBeUndefined()
    expect(job.started_at).toBeUndefined()
    expect(job.completed_at).toBeUndefined()
  })
})

describe('isRunning helper', () => {
  const isRunning = (status: BacktestJob['status']): boolean =>
    status === 'pending' || status === 'downloading' || status === 'running'

  it('returns true for active statuses', () => {
    expect(isRunning('pending')).toBe(true)
    expect(isRunning('downloading')).toBe(true)
    expect(isRunning('running')).toBe(true)
  })

  it('returns false for terminal statuses', () => {
    expect(isRunning('completed')).toBe(false)
    expect(isRunning('failed')).toBe(false)
  })
})

describe('stripAnsi helper', () => {
  const stripAnsi = (s: string): string => s.replace(/\x1b\[[0-9;]*m/g, '')

  it('removes ANSI escape codes from backtest output', () => {
    const output = '\x1b[32mProfit: 1234.5\x1b[0m\x1b[33mWarning: low volume\x1b[0m'
    expect(stripAnsi(output)).toBe('Profit: 1234.5Warning: low volume')
  })

  it('returns clean string unchanged', () => {
    expect(stripAnsi('clean output')).toBe('clean output')
  })

  it('handles empty string', () => {
    expect(stripAnsi('')).toBe('')
  })
})
