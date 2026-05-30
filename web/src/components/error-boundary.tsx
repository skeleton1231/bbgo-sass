'use client'

import { Component } from 'react'
import { useTranslations } from 'next-intl'

interface Props {
  children: React.ReactNode
  fallback?: React.ReactNode
}

interface State {
  hasError: boolean
}

function ErrorFallback({ onRetry }: { onRetry: () => void }) {
  const t = useTranslations('Errors')
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border bg-card p-8 text-center">
      <p className="text-sm text-muted-foreground">{t('componentError')}</p>
      <button
        type="button"
        onClick={onRetry}
        className="mt-3 rounded-md bg-primary px-4 py-1.5 text-sm text-primary-foreground hover:bg-primary/90"
      >
        {t('retry')}
      </button>
    </div>
  )
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <ErrorFallback onRetry={() => this.setState({ hasError: false })} />
      )
    }
    return this.props.children
  }
}
