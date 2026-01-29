import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'
import { CodeBlock } from '~/components/CodeBlock'
import type { SessionDetail } from '~/utils/localSessions'

export const Route = createFileRoute('/sessions/$sessionId')({
  component: SessionDetailView,
})

function SessionDetailView() {
  const { sessionId } = Route.useParams()
  const [session, setSession] = React.useState<SessionDetail | null>(null)
  const [config, setConfig] = React.useState<any>(null)
  const [loading, setLoading] = React.useState(true)
  const [loadError, setLoadError] = React.useState('')

  React.useEffect(() => {
    let active = true
    setLoading(true)
    setLoadError('')
    Promise.all([
      fetch(`/api/sessions/${sessionId}`),
      fetch('/api/config').catch(() => null),
    ])
      .then(async ([sessionRes, configRes]) => {
        if (!sessionRes.ok) {
          throw new Error('Session not found')
        }
        const sessionData = (await sessionRes.json()) as { session: SessionDetail }
        const configData = configRes && configRes.ok ? await configRes.json() : null
        if (!active) return
        setSession(sessionData.session)
        setConfig(configData)
        setLoading(false)
      })
      .catch((err: Error) => {
        if (!active) return
        setLoadError(err.message || 'Failed to load session')
        setLoading(false)
      })
    return () => {
      active = false
    }
  }, [sessionId])

  const defaultTags = Array.isArray(config?.remote?.default_tags)
    ? (config.remote.default_tags as string[])
    : []
  const defaultTagString = defaultTags.join(', ')
  const [shareOpen, setShareOpen] = React.useState(false)
  const [shareTags, setShareTags] = React.useState(defaultTagString)
  const [shareStatus, setShareStatus] = React.useState('')

  React.useEffect(() => {
    if (!shareOpen) {
      setShareStatus('')
      return
    }
    if (!shareTags && defaultTagString) {
      setShareTags(defaultTagString)
    }
  }, [shareOpen, shareTags, defaultTagString])

  if (loading) {
    return <div className="loading">Loading session...</div>
  }

  if (loadError || !session) {
    return (
      <main className="main">
        <header className="session-header">
          <Link className="back-link" to="/">
            ← Back
          </Link>
          <div className="session-id">Session unavailable</div>
        </header>
        <div className="empty-state">
          <div className="icon">Alert</div>
          <h2>Unable to load session</h2>
          <p>{loadError || 'Session not found'}</p>
        </div>
      </main>
    )
  }

  const messageCount = session.events.filter((e) => e.event_type === 'message').length
  const toolCount = session.events.filter((e) => e.event_type === 'tool_use').length

  const items = buildItems(session.events)

  const handleShare = async () => {
    setShareStatus('Sharing...')
    const tags = parseTags(shareTags)
    const resp = await fetch('/api/sessions/push', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: session.session_id, tool: session.tool, tags }),
    })
    if (resp.ok) {
      const payload = await resp.json().catch(() => null)
      setShareStatus(payload?.url ? `Shared: ${payload.url}` : 'Shared!')
      setTimeout(() => setShareOpen(false), 1200)
    } else {
      const payload = await resp.json().catch(() => null)
      setShareStatus(payload?.error?.message || 'Share failed')
    }
  }

  return (
    <main className="main">
      <header className="session-header">
        <Link className="back-link" to="/">
          ← Back
        </Link>
        <div className="session-id">Session: {session.session_id}</div>
        <div className="session-actions">
          <button className="primary-btn" onClick={() => setShareOpen(true)}>
            Share
          </button>
        </div>
        <div className="session-meta-line">
          {session.tool} · {session.created_at}
        </div>
        <div className="session-meta-line">{session.cwd}</div>
        <div className="session-meta-line">
          {messageCount} messages · {toolCount} tools
        </div>
      </header>

      <div className="detail-list">
        {items.map((item, idx) => {
          if (item.type === 'message') {
            return (
              <div className="message-card" key={idx}>
                <div className="role">{item.role === 'user' ? 'User' : 'Assistant'}</div>
                {item.content ? <div className="message-content">{item.content}</div> : null}
                {item.thinking?.length ? (
                  <details className="thinking-block">
                    <summary>Thinking</summary>
                    <div className="thinking-content">{item.thinking.join('\n\n')}</div>
                  </details>
                ) : null}
              </div>
            )
          }
          return (
            <div className={`tool-card${item.is_error ? ' error' : ''}`} key={idx}>
              <div className="tool-title">
                <span>Tool {item.tool_name}</span>
                <span>{item.timestamp}</span>
              </div>
              {item.input ? (
                <details open>
                  <summary>Input</summary>
                  <CodeBlock code={formatCode(item.input)} language="json" />
                </details>
              ) : null}
              {item.output ? (
                <details open>
                  <summary>Output</summary>
                  <CodeBlock code={formatCode(item.output)} language="json" />
                </details>
              ) : null}
            </div>
          )
        })}
      </div>

      {shareOpen ? (
        <div className="modal-overlay" onClick={() => setShareOpen(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Share Session</h3>
              <button className="ghost-btn" onClick={() => setShareOpen(false)}>
                Close
              </button>
            </div>
            <p className="modal-subtext">Add tags like team:platform, repo:myapp.</p>
            <label className="filter-label">
              Tags (optional)
              <input
                className="search-input"
                value={shareTags}
                onChange={(e) => setShareTags(e.target.value)}
                placeholder="team:platform, repo:myapp"
              />
            </label>
            <div className="session-stats">{shareStatus}</div>
            <div className="modal-actions">
              <button className="ghost-btn" onClick={() => setShareOpen(false)}>
                Cancel
              </button>
              <button className="primary-btn" onClick={handleShare}>
                Share →
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </main>
  )
}

function buildItems(events: Record<string, any>[]) {
  const items: any[] = []
  const toolMap = new Map<string, any>()
  events.forEach((event) => {
    const data = event.data || {}
    if (event.event_type === 'message') {
      const parts = extractMessageParts(data.content || data.text || '')
      items.push({ type: 'message', role: data.role, content: parts.text, thinking: parts.thinking })
    }
    if (event.event_type === 'tool_use') {
      const item = {
        type: 'tool',
        tool_use_id: data.tool_use_id,
        tool_name: data.tool_name,
        input: data.input,
        output: null,
        is_error: false,
        timestamp: event.timestamp,
      }
      items.push(item)
      if (data.tool_use_id) toolMap.set(data.tool_use_id, item)
    }
    if (event.event_type === 'tool_result') {
      const existing = toolMap.get(data.tool_use_id)
      if (existing) {
        existing.output = data.content
        existing.is_error = Boolean(data.is_error)
      } else {
        items.push({
          type: 'tool',
          tool_use_id: data.tool_use_id,
          tool_name: 'tool',
          input: null,
          output: data.content,
          is_error: Boolean(data.is_error),
          timestamp: event.timestamp,
        })
      }
    }
  })
  return items
}

function extractMessageParts(content: any) {
  const text: string[] = []
  const thinking: string[] = []
  if (!content) return { text: '', thinking: [] }
  if (typeof content === 'string') {
    text.push(content)
    return { text: text.join('\n'), thinking }
  }
  if (Array.isArray(content)) {
    content.forEach((part) => {
      if (typeof part === 'string') {
        text.push(part)
        return
      }
      if (!part || typeof part !== 'object') return
      const value = part.text || part.content || ''
      if (!value) return
      const kind = String(part.type || '').toLowerCase()
      if (kind === 'thinking' || kind === 'thought') {
        thinking.push(value)
      } else {
        text.push(value)
      }
    })
    return { text: text.join('\n'), thinking }
  }
  if (typeof content === 'object') {
    const value = content.text || content.content || ''
    if (value) {
      const kind = String(content.type || '').toLowerCase()
      if (kind === 'thinking' || kind === 'thought') {
        thinking.push(value)
      } else {
        text.push(value)
      }
    }
  }
  return { text: text.join('\n'), thinking }
}

function formatCode(value: any) {
  if (value == null) return ''
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function parseTags(raw: string) {
  return raw
    .split(/[,\n]/)
    .map((part) => part.trim())
    .filter(Boolean)
    .map((token) => {
      const [key, ...rest] = token.includes(':') ? token.split(':') : token.split('=')
      return { key: key.trim(), value: rest.join(':').trim() }
    })
    .filter((tag) => tag.key && tag.value)
}
