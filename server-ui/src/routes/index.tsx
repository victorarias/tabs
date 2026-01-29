import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'

export type TagCount = { key: string; value: string; count: number }
export type SessionSummary = {
  id: string
  tool: string
  session_id: string
  created_at: string
  ended_at?: string
  cwd?: string
  uploaded_by: string
  uploaded_at: string
  duration_seconds?: number
  message_count: number
  tool_use_count: number
  tags: { key: string; value: string }[]
  summary?: string
}

export const Route = createFileRoute('/')({
  validateSearch: (search) => {
    const toString = (value: unknown) => (typeof value === 'string' ? value : undefined)
    const toNumber = (value: unknown) => {
      if (typeof value === 'string' && value.trim() !== '') {
        const parsed = Number(value)
        if (!Number.isNaN(parsed) && parsed > 0) return Math.floor(parsed)
      }
      if (typeof value === 'number' && value > 0) return Math.floor(value)
      return undefined
    }
    const tags = Array.isArray(search.tag)
      ? search.tag.filter((value) => typeof value === 'string')
      : typeof search.tag === 'string'
      ? [search.tag]
      : []
    return {
      q: toString(search.q),
      tool: toString(search.tool),
      tag: tags.length ? tags : undefined,
      sort: toString(search.sort),
      order: toString(search.order),
      page: toNumber(search.page),
      limit: toNumber(search.limit),
    }
  },
  component: SharedTimeline,
})

function SharedTimeline() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const [queryInput, setQueryInput] = React.useState(search.q ?? '')
  const tool = search.tool ?? ''
  const activeTags = search.tag ?? []
  const sort = search.sort ?? 'created_at'
  const order = search.order ?? 'desc'
  const page = search.page ?? 1
  const limit = search.limit ?? 20
  const [sessions, setSessions] = React.useState<SessionSummary[]>([])
  const [tags, setTags] = React.useState<TagCount[]>([])
  const [pagination, setPagination] = React.useState<{
    page: number
    limit: number
    total: number
    total_pages: number
  } | null>(null)
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')

  const buildSearch = React.useCallback(
    (overrides: Partial<typeof search>) => ({
      q: search.q,
      tool: search.tool,
      tag: search.tag,
      sort: search.sort,
      order: search.order,
      page: search.page,
      limit: search.limit,
      ...overrides,
    }),
    [search.q, search.tool, search.tag, search.sort, search.order, search.page, search.limit],
  )

  React.useEffect(() => {
    setQueryInput(search.q ?? '')
  }, [search.q])

  React.useEffect(() => {
    let active = true
    setLoading(true)
    setError('')
    const params = new URLSearchParams()
    if (search.q) params.set('q', search.q)
    if (search.tool) params.set('tool', search.tool)
    if (search.sort) params.set('sort', search.sort)
    if (search.order) params.set('order', search.order)
    if (search.page) params.set('page', String(search.page))
    if (search.limit) params.set('limit', String(search.limit))
    if (Array.isArray(search.tag)) {
      search.tag.forEach((tag) => params.append('tag', tag))
    }

    fetch(`/api/sessions?${params.toString()}`)
      .then(async (res) => {
        if (!res.ok) throw new Error('Failed to load sessions')
        const payload = (await res.json()) as {
          sessions: SessionSummary[]
          pagination?: { page: number; limit: number; total: number; total_pages: number }
        }
        if (!active) return
        setSessions(payload.sessions || [])
        setPagination(payload.pagination || null)
        setLoading(false)
      })
      .catch((err: Error) => {
        if (!active) return
        setError(err.message || 'Failed to load sessions')
        setLoading(false)
      })

    return () => {
      active = false
    }
  }, [search.q, search.tool, search.tag, search.sort, search.order, search.page, search.limit])

  React.useEffect(() => {
    let active = true
    fetch('/api/tags?limit=20')
      .then(async (res) => {
        if (!res.ok) return { tags: [] as TagCount[] }
        const payload = (await res.json()) as { tags: TagCount[] }
        if (!active) return
        setTags(payload.tags || [])
      })
      .catch(() => {})
    return () => {
      active = false
    }
  }, [])

  React.useEffect(() => {
    if (queryInput === (search.q ?? '')) return
    const timer = setTimeout(() => {
      navigate({
        to: '/',
        search: buildSearch({ q: queryInput || undefined, page: 1 }),
      })
    }, 250)
    return () => clearTimeout(timer)
  }, [queryInput, search.q, navigate, buildSearch])

  const grouped = groupByDate(sessions)
  const tools = React.useMemo(() => {
    return Array.from(new Set(sessions.map((s) => s.tool).filter(Boolean))).sort()
  }, [sessions])

  return (
    <main className="main">
      <header className="app-header">
        <div className="logo" role="link">
          <span className="logo-text">tabs</span>
          <span className="logo-sub">shared</span>
        </div>
        <div className="search-wrap" role="search">
          <input
            className="search-input"
            placeholder="Search across shared sessions..."
            value={queryInput}
            onChange={(e) => setQueryInput(e.target.value)}
          />
        </div>
        <div className="header-actions">
          <Link className="nav-link" to="/" activeOptions={{ exact: true }}>
            Sessions
          </Link>
          <Link className="nav-link" to="/keys">
            Keys
          </Link>
        </div>
      </header>

      <section className="toolbar">
        <div className="filters">
          <label className="filter-label">
            Tool
            <select
              className="filter-select"
              value={tool}
              onChange={(e) =>
                navigate({
                  to: '/',
                  search: buildSearch({ tool: e.target.value || undefined, page: 1 }),
                })
              }
            >
              <option value="">All</option>
              {tools.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </label>
          <label className="filter-label">
            Sort
            <select
              className="filter-select"
              value={`${sort}:${order}`}
              onChange={(e) => {
                const [nextSort, nextOrder] = e.target.value.split(':')
                navigate({
                  to: '/',
                  search: buildSearch({
                    sort: nextSort || undefined,
                    order: nextOrder || undefined,
                    page: 1,
                  }),
                })
              }}
            >
              <option value="created_at:desc">Newest</option>
              <option value="created_at:asc">Oldest</option>
              <option value="uploaded_at:desc">Recently shared</option>
              <option value="uploaded_at:asc">Oldest shared</option>
            </select>
          </label>
          <div className="filter-label tag-filter">
            <span>Popular tags</span>
            <div className="tag-cloud">
              {tags.slice(0, 10).map((tag) => {
                const key = `${tag.key}:${tag.value}`
                const active = activeTags.includes(key)
                return (
                  <button
                    key={`${tag.key}:${tag.value}`}
                    className={`tag-pill${active ? ' active' : ''}`}
                    onClick={() => {
                      const next = active
                        ? activeTags.filter((value) => value !== key)
                        : [...activeTags, key]
                      navigate({
                        to: '/',
                        search: buildSearch({ tag: next.length ? next : undefined, page: 1 }),
                      })
                    }}
                  >
                    {tag.key}:{tag.value} ({tag.count})
                  </button>
                )
              })}
            </div>
          </div>
        </div>
        <div className="session-count">
          {loading ? 'Loading…' : `${pagination?.total ?? sessions.length} sessions`}
        </div>
      </section>

      {activeTags.length > 0 ? (
        <div className="active-tags">
          {activeTags.map((tag) => (
            <span key={tag} className="active-tag">
              {tag}
              <button
                onClick={() =>
                navigate({
                  to: '/',
                  search: buildSearch({
                    tag: activeTags.filter((value) => value !== tag),
                    page: 1,
                  }),
                })
              }
            >
                ×
              </button>
            </span>
          ))}
        </div>
      ) : null}

      {error ? (
        <div className="empty-state">
          <div className="icon">Alert</div>
          <h2>Unable to load sessions</h2>
          <p>{error}</p>
        </div>
      ) : sessions.length === 0 ? (
        <div className="empty-state">
          <div className="icon">Tabs</div>
          <h2>No sessions shared yet</h2>
          <p>Be the first to share a session!</p>
        </div>
      ) : (
        grouped.map(([day, items]) => (
          <section className="timeline-group" key={day}>
            <div className="timeline-date">{formatDayLabel(day)}</div>
            {items.map((session) => (
              <Link
                className="session-card"
                key={session.id}
                to="/sessions/$sessionId"
                params={{ sessionId: session.id }}
              >
                <div className="session-meta">
                  <span>{formatTime(session.created_at)}</span>
                  <span className="tool-badge">{session.tool}</span>
                  <span>{formatDuration(session.duration_seconds ?? 0)}</span>
                </div>
                <div className="session-title">
                  {session.summary || `Session ${session.session_id}`}
                </div>
                <div className="session-stats">{session.message_count} messages · {session.tool_use_count} tools</div>
                <div className="session-meta-line">{session.uploaded_by}</div>
              </Link>
            ))}
          </section>
        ))
      )}

      {pagination ? (
        <div className="pagination">
          <button
            className="ghost-btn"
            disabled={page <= 1}
            onClick={() =>
              navigate({ to: '/', search: buildSearch({ page: Math.max(1, page - 1) }) })
            }
          >
            Prev
          </button>
          <div className="pagination-meta">
            Page {page} of {pagination.total_pages}
          </div>
          <button
            className="ghost-btn"
            disabled={page >= pagination.total_pages}
            onClick={() =>
              navigate({
                to: '/',
                search: buildSearch({ page: Math.min(pagination.total_pages, page + 1) }),
              })
            }
          >
            Next
          </button>
          <label className="filter-label">
            Per page
            <select
              className="filter-select"
              value={limit}
              onChange={(e) =>
                navigate({
                  to: '/',
                  search: buildSearch({ limit: Number(e.target.value), page: 1 }),
                })
              }
            >
              {[10, 20, 50, 100].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </label>
        </div>
      ) : null}
    </main>
  )
}

function groupByDate(sessions: SessionSummary[]) {
  const groups = new Map<string, SessionSummary[]>()
  sessions.forEach((session) => {
    const key = session.created_at.slice(0, 10)
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
