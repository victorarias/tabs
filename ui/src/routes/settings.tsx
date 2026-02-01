import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'

export const Route = createFileRoute('/settings')({
  component: Settings,
})

function Settings() {
  const [data, setData] = React.useState<any>(null)
  const [loading, setLoading] = React.useState(true)
  const [loadError, setLoadError] = React.useState('')
  const [serverUrl, setServerUrl] = React.useState('')
  const [apiKey, setApiKey] = React.useState('')
  const [defaultTags, setDefaultTags] = React.useState('')
  const [status, setStatus] = React.useState('')

  React.useEffect(() => {
    let active = true
    setLoading(true)
    setLoadError('')
    fetch('/api/config')
      .then(async (res) => {
        if (!res.ok) throw new Error('Failed to load config')
        const payload = await res.json()
        if (!active) return
        setData(payload)
        setServerUrl(payload.remote?.server_url || '')
        setDefaultTags(
          Array.isArray(payload.remote?.default_tags) ? payload.remote.default_tags.join(', ') : '',
        )
        setLoading(false)
      })
      .catch((err: Error) => {
        if (!active) return
        setLoadError(err.message || 'Failed to load config')
        setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const apiKeyConfigured = Boolean(data?.remote?.api_key_configured)
  const apiKeyPrefix = data?.remote?.api_key_prefix || ''

  const save = async () => {
    setStatus('Saving...')
    const payload: Record<string, any> = {
      remote: {
        server_url: serverUrl,
        default_tags: defaultTags,
      },
    }
    if (apiKey.trim()) {
      payload.remote.api_key = apiKey
    }
    const resp = await fetch('/api/config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (resp.ok) {
      setStatus('Saved.')
      setApiKey('')
    } else {
      const payload = await resp.json().catch(() => null)
      setStatus(payload?.error?.message || 'Failed to save')
    }
  }

  return (
    <main className="main">
      <header className="session-header">
        <Link className="back-link" to="/">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M19 12H5M12 19l-7-7 7-7" />
          </svg>
          Back to sessions
        </Link>
        <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 'var(--text-h2)', letterSpacing: '0.1em', margin: 0 }}>
          SETTINGS
        </h2>
      </header>
      {loading ? (
        <div className="loading">Initializing...</div>
      ) : loadError ? (
        <div className="empty-state">
          <div className="icon">!</div>
          <h2>Configuration error</h2>
          <p>{loadError}</p>
        </div>
      ) : (
        <section className="settings-card">
          <div className="role">Remote Server</div>
          <label className="filter-label">
            Server URL
            <input
              value={serverUrl}
              onChange={(e) => setServerUrl(e.target.value)}
              placeholder="https://tabs.example.com"
            />
          </label>
          <label className="filter-label">
            API Key
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="tabs_..."
            />
          </label>
          <div className="session-stats" style={{ marginTop: 'var(--space-2)' }}>
            {apiKeyConfigured ? (
              <>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--success)" strokeWidth="2" style={{ marginRight: '6px' }}>
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
                Key configured ({apiKeyPrefix}...)
              </>
            ) : (
              <>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--fg-tertiary)" strokeWidth="2" style={{ marginRight: '6px' }}>
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="8" x2="12" y2="12" />
                  <line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
                No API key configured
              </>
            )}
          </div>
          <label className="filter-label">
            Default Tags
            <input
              value={defaultTags}
              onChange={(e) => setDefaultTags(e.target.value)}
              placeholder="team:platform, repo:myapp"
            />
          </label>
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-4)', marginTop: 'var(--space-4)' }}>
            <button className="primary-btn" onClick={save}>
              Save Changes
            </button>
            {status && (
              <span className="session-stats" style={{ margin: 0 }}>
                {status}
              </span>
            )}
          </div>
        </section>
      )}
    </main>
  )
}
