import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'

export const Route = createFileRoute('/keys')({
  loader: async ({ request }) => {
    const res = await fetch(apiURL('/api/keys', request))
    if (!res.ok) throw new Error('Failed to load keys')
    return res.json() as Promise<any>
  },
  component: KeysPage,
})

function KeysPage() {
  const data = Route.useLoaderData()
  const [name, setName] = React.useState('')
  const [status, setStatus] = React.useState('')
  const [createdKey, setCreatedKey] = React.useState('')

  const createKey = async () => {
    if (!name.trim()) {
      setStatus('Name is required.')
      return
    }
    setStatus('Creating...')
    setCreatedKey('')
    const res = await fetch('/api/keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    if (!res.ok) {
      const payload = await res.json().catch(() => null)
      setStatus(payload?.error?.message || 'Failed to create key.')
      return
    }
    const payload = await res.json().catch(() => null)
    setStatus('Key created. Copy it now — it will only be shown once.')
    setCreatedKey(payload?.key || '')
    setName('')
  }

  const revoke = async (id: string) => {
    const res = await fetch(`/api/keys/${id}`, { method: 'DELETE' })
    if (res.ok) {
      location.reload()
    }
  }

  return (
    <main className="main">
      <header className="session-header">
        <Link className="back-link" to="/">
          ← Back
        </Link>
        <h2>API Keys</h2>
      </header>
      <section className="message-card keys-card">
        <div className="role">Create new key</div>
        <label className="filter-label">
          Name
          <input
            className="search-input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My laptop"
          />
        </label>
        <button className="ghost-btn" onClick={createKey}>
          Create Key
        </button>
        <div className="session-stats">{status}</div>
        {createdKey ? <div className="session-stats">{createdKey}</div> : null}
      </section>
      <section className="message-card keys-card">
        <div className="role">Existing keys</div>
        <div>
          {(data.keys || []).map((key: any) => (
            <div className="key-row" key={key.id}>
              <div>
                <div className="key-name">{key.name}</div>
                <div className="session-meta-line">{key.key_prefix}</div>
              </div>
              <div className="key-meta">
                <div className="session-meta-line">{key.is_active ? 'Active' : 'Revoked'}</div>
                <button className="ghost-btn" disabled={!key.is_active} onClick={() => revoke(key.id)}>
                  Revoke
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}

function apiURL(path: string, request?: Request) {
  if (!request) return path
  const host = request.headers.get('host') || 'localhost'
  const proto = request.headers.get('x-forwarded-proto') || 'http'
  return new URL(path, `${proto}://${host}`).toString()
}
