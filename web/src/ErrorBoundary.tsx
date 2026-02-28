import React from 'react'

type Props = {
  children: React.ReactNode
}

type State = {
  error: Error | null
  errorInfo: React.ErrorInfo | null
}

export default class ErrorBoundary extends React.Component<Props, State> {
  state: State = { error: null, errorInfo: null }

  static getDerivedStateFromError(error: Error): State {
    return { error, errorInfo: null }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Never include localStorage or any secrets; this is UI-only.
    this.setState({ error, errorInfo })
  }

  render() {
    if (!this.state.error) return this.props.children

    const msg = this.state.error?.message || 'Unknown error'
    const dev = import.meta.env.DEV
    const stack = dev ? this.state.error?.stack : ''
    const componentStack = dev ? this.state.errorInfo?.componentStack : ''

    return (
      <div style={{ padding: 16 }}>
        <h2 style={{ marginTop: 0 }}>Something went wrong</h2>
        <p style={{ color: '#fbb' }}>{msg}</p>
        <p style={{ color: '#9aa3b2' }}>
          This page should never be completely blank. If this repeats, refresh once. If it persists, run{' '}
          <code>./scripts/collect_diag_bundle_v6.sh</code> and attach the bundle.
        </p>
        {dev && (
          <pre style={{ whiteSpace: 'pre-wrap', background: '#0d0d14', border: '1px solid #2a2a3a', padding: 12 }}>
            {stack}
            {componentStack}
          </pre>
        )}
        <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
          <button type="button" onClick={() => window.location.reload()}>
            Reload
          </button>
        </div>
      </div>
    )
  }
}

