import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'
import { CodeBlock } from '~/components/CodeBlock'

export const Route = createFileRoute('/sessions/$sessionId')({
  component: SessionDetail,
})

function SessionDetail() {
  const { sessionId } = Route.useParams()
  const [session, setSession] = React.useState<any>(null)
  const [loading, setLoading] = React.useState(true)
  const [loadError, setLoadError] = React.useState('')

  React.useEffect(() => {
    let active = true
    setLoading(true)
    setLoadError('')
    fetch(`/api/sessions/${sessionId}`)
      .then(async (res) => {
        if (!res.ok) throw new Error('Session not found')
        const payload = (await res.json()) as { session: any }
        if (!active) return
        setSession(payload.session)
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

  const items = buildItems(session)

  return (
    <main className="main">
      <header className="session-header">
        <Link className="back-link" to="/">
          ← Back
        </Link>
        <div className="session-id">Session: {session.session_id}</div>
        <div className="session-meta-line">{session.tool}</div>
        <div className="session-meta-line">{session.cwd}</div>
        <div className="session-meta-line">Uploaded by {session.uploaded_by}</div>
      </header>
      <div className="detail-list">
        {items.map((item: any, idx: number) =>
          item.type === 'message' ? (
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
          ) : (
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
          ),
        )}
      </div>
    </main>
  )
}

function buildItems(session: any) {
  const items: any[] = []
  session.messages?.forEach((msg: any) => {
    const parts = extractMessageParts(msg.content)
    items.push({ type: 'message', role: msg.role, content: parts.text, thinking: parts.thinking })
  })
  session.tools?.forEach((tool: any) => {
    items.push({
      type: 'tool',
      tool_name: tool.tool_name,
      input: tool.input,
      output: tool.output,
      is_error: tool.is_error,
      timestamp: tool.timestamp,
    })
  })
  items.sort((a, b) => new Date(a.timestamp || 0).getTime() - new Date(b.timestamp || 0).getTime())
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
