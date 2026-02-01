import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'

type SessionSummary = {
  session_id: string
  tool: string
  created_at: string
  ended_at?: string
  cwd?: string
  summary?: string
  duration_seconds?: number
  message_count: number
  tool_use_count: number
}

export const Route = createFileRoute('/')({
  component: Timeline,
})

function Timeline() {
  const [query, setQuery] = React.useState('')
  const [tool, setTool] = React.useState('')
  const [date, setDate] = React.useState('')
  const [cwd, setCwd] = React.useState('')
  const [sessions, setSessions] = React.useState<SessionSummary[]>([])
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState('')

  const [toolOptions, setToolOptions] = React.useState<string[]>([])

  React.useEffect(() => {
    setToolOptions((prev) => {
      const next = new Set(prev)
      sessions.forEach((session) => {
        if (session.tool) next.add(session.tool)
      })
      const sorted = Array.from(next).sort()
      if (sorted.length === prev.length && sorted.every((value, idx) => value === prev[idx])) {
        return prev
      }
      return sorted
    })
  }, [sessions])

  React.useEffect(() => {
    let active = true
    const controller = new AbortController()
    const fetchSessions = async () => {
      setLoading(true)
      setError('')
      const params = new URLSearchParams()
      if (query) params.set('q', query)
      if (tool) params.set('tool', tool)
      if (date) params.set('date', date)
      if (cwd) params.set('cwd', cwd)
      const url = `/api/sessions${params.toString() ? `?${params.toString()}` : ''}`
      const res = await fetch(url, { signal: controller.signal })
      if (!res.ok) throw new Error('Failed to load sessions')
      const payload = (await res.json()) as { sessions?: SessionSummary[] }
      if (!active) return
      setSessions(payload.sessions || [])
      setLoading(false)
    }

    const delay = query ? 200 : 0
    const timer = setTimeout(() => {
      fetchSessions().catch((err: Error) => {
        if (!active) return
        if (err.name === 'AbortError') return
        setError(err.message || 'Failed to load sessions')
        setLoading(false)
      })
    }, delay)

    return () => {
      active = false
      controller.abort()
      clearTimeout(timer)
    }
  }, [query, tool, date, cwd])

  const grouped = groupByDate(sessions)

  return (
    <main className="main">
      <header className="app-header">
        <Link className="logo" to="/">
          <span className="logo-text">TABS</span>
          <span className="logo-sub">local</span>
        </Link>
        <div className="search-wrap" role="search">
          <svg className="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            className="search-input"
            placeholder="Search sessions, messages, files..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {query && (
            <button className="clear-btn visible" onClick={() => setQuery('')} aria-label="Clear search">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M18 6 6 18M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
        <nav className="header-actions">
          <Link className="nav-link" to="/" activeOptions={{ exact: true }}>
            Sessions
          </Link>
          <Link className="nav-link" to="/settings">
            Settings
          </Link>
        </nav>
      </header>

      <section className="toolbar">
        <div className="filters">
          <label className="filter-label">
            Tool
            <select
              className="filter-select"
              value={tool}
              onChange={(e) => setTool(e.target.value)}
            >
              <option value="">All tools</option>
              {toolOptions.map((t) => (
                <option key={t} value={t}>
                  {formatToolName(t)}
                </option>
              ))}
            </select>
          </label>
          <label className="filter-label">
            Date
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </label>
          <label className="filter-label">
            Directory
            <input
              value={cwd}
              onChange={(e) => setCwd(e.target.value)}
              placeholder="~/projects/..."
            />
          </label>
        </div>
        <div className="session-count">
          {loading ? 'scanning...' : `${sessions.length} sessions`}
        </div>
      </section>

      {error ? (
        <div className="empty-state">
          <div className="icon">!</div>
          <h2>Connection failed</h2>
          <p>{error}</p>
        </div>
      ) : sessions.length === 0 ? (
        <div className="empty-state">
          <div className="icon">_</div>
          <h2>No sessions found</h2>
          <p>Start a session with Claude Code or Cursor to see your history here.</p>
        </div>
      ) : (
        grouped.map(([day, items]) => (
          <section className="timeline-group" key={day}>
            <h2 className="timeline-date">{formatDayLabel(day)}</h2>
            {items.map((session, idx) => (
              <Link
                className="session-card note-in"
                key={session.session_id}
                to="/sessions/$sessionId"
                params={{ sessionId: session.session_id }}
                style={{ animationDelay: `${Math.min(idx * 50, 400)}ms` }}
              >
                <div className="session-meta">
                  <time>{formatTime(session.created_at || session.ended_at || '')}</time>
                  <span className="tool-badge" data-tool={getToolId(session.tool)}>
                    {formatToolName(session.tool)}
                  </span>
                  <span>{formatDuration(session.duration_seconds ?? 0)}</span>
                </div>
                {session.cwd && (
                  <div className="session-path">{shortenPath(session.cwd)}</div>
                )}
                <div className="session-title">
                  {session.summary || `Session ${session.session_id.slice(0, 8)}`}
                </div>
                <div className="session-stats">
                  <span>{session.message_count} msgs</span>
                  <span>{session.tool_use_count} tools</span>
                </div>
              </Link>
            ))}
          </section>
        ))
      )}
    </main>
  )
}

function groupByDate(sessions: SessionSummary[]) {
  const groups = new Map<string, SessionSummary[]>()
  sessions.forEach((session) => {
    const key = (session.created_at || session.ended_at || '').slice(0, 10)
    const list = groups.get(key) || []
    list.push(session)
    groups.set(key, list)
  })
  return Array.from(groups.entries())
}

function formatTime(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
}

function formatDayLabel(value: string) {
  if (!value) return 'Unknown'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' })
}

function formatDuration(seconds: number) {
  if (!seconds) return '--'
  const mins = Math.round(seconds / 60)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  const rem = mins % 60
  return rem ? `${hours}h ${rem}m` : `${hours}h`
}

function shortenPath(value: string) {
  const parts = value.split('/').filter(Boolean)
  if (parts.length <= 3) return value
  return `.../${parts.slice(-3).join('/')}`
}

function formatToolName(tool: string) {
  const t = tool?.toLowerCase().replace(/-/g, '_') || ''
  if (t === 'claude_code') return 'Claude'
  if (t === 'cursor') return 'Cursor'
  // Capitalize first letter of each word
  return tool?.replace(/[-_]/g, ' ').replace(/\b\w/g, c => c.toUpperCase()) || ''
}

function getToolId(tool: string) {
  // Normalize to underscore format for CSS selectors
  return tool?.toLowerCase().replace(/[-\s]+/g, '_') || ''
}
