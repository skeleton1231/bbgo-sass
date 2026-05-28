'use client'

import { Component } from 'react'

interface Props {
  children: React.ReactNode
  fallback?: React.ReactNode
}

interface State {
  hasError: boolean
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback ?? (
        <div className="flex flex-col items-center justify-center rounded-lg border bg-card p-8 text-center">
          <p className="text-sm text-muted-foreground">Something went wrong. Refresh the page to try again.</p>
          <button
            type="button"
            onClick={() => this.setState({ hasError: false })}
            className="mt-3 rounded-md bg-primary px-4 py-1.5 text-sm text-primary-foreground hover:bg-primary/90"
          >
            Retry
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
