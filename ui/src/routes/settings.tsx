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
          ← Back
        </Link>
        <h2>Settings</h2>
      </header>
      {loading ? (
        <div className="loading">Loading settings...</div>
      ) : loadError ? (
        <div className="empty-state">
          <div className="icon">Alert</div>
          <h2>Unable to load settings</h2>
          <p>{loadError}</p>
        </div>
      ) : (
      <section className="message-card">
        <div className="role">Remote Server</div>
        <label className="filter-label">
          Server URL
          <input
            className="search-input"
            value={serverUrl}
            onChange={(e) => setServerUrl(e.target.value)}
          />
        </label>
        <label className="filter-label">
          API Key
          <input
            className="search-input"
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="tabs_..."
          />
        </label>
        {apiKeyConfigured ? (
          <div className="session-stats">API key configured ({apiKeyPrefix}…)</div>
        ) : (
          <div className="session-stats">No API key configured</div>
        )}
        <label className="filter-label">
          Default Tags
          <input
            className="search-input"
            value={defaultTags}
            onChange={(e) => setDefaultTags(e.target.value)}
            placeholder="team:platform, repo:myapp"
          />
        </label>
        <button className="ghost-btn" onClick={save}>
          Save Changes
        </button>
        <div className="session-stats">{status}</div>
      </section>
      )}
    </main>
  )
}
