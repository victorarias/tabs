import { createFileRoute, Link } from '@tanstack/react-router'
import * as React from 'react'
import { CodeBlock } from '~/components/CodeBlock'

type SessionDetail = {
  session_id: string
  tool: string
  created_at: string
  ended_at?: string
  cwd?: string
  duration_seconds?: number
  events: Record<string, unknown>[]
}

// Get language from file extension for syntax highlighting
function getLanguageFromPath(filePath: string): string {
  const ext = filePath.split('.').pop()?.toLowerCase() || ''
  const langMap: Record<string, string> = {
    js: 'javascript',
    jsx: 'jsx',
    ts: 'typescript',
    tsx: 'tsx',
    py: 'python',
    rb: 'ruby',
    go: 'go',
    rs: 'rust',
    java: 'java',
    kt: 'kotlin',
    swift: 'swift',
    c: 'c',
    cpp: 'cpp',
    h: 'c',
    hpp: 'cpp',
    cs: 'csharp',
    php: 'php',
    html: 'html',
    css: 'css',
    scss: 'scss',
    less: 'less',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    xml: 'xml',
    md: 'markdown',
    sql: 'sql',
    sh: 'bash',
    bash: 'bash',
    zsh: 'bash',
    fish: 'fish',
    ps1: 'powershell',
    dockerfile: 'dockerfile',
    makefile: 'makefile',
    toml: 'toml',
    ini: 'ini',
    conf: 'ini',
    env: 'dotenv',
    gitignore: 'gitignore',
    vue: 'vue',
    svelte: 'svelte',
    astro: 'astro',
  }
  return langMap[ext] || 'text'
}

// Strip system-reminder tags from content
function stripSystemReminders(content: string): string {
  // Remove <system-reminder>...</system-reminder> blocks
  return content.replace(/<system-reminder>[\s\S]*?<\/system-reminder>/gi, '').trim()
}

// Strip line number prefixes from file output (e.g., "  1→content" -> "content")
function stripLineNumbers(content: string): string {
  // Match patterns like "     1→" or "  123→" at the start of lines
  return content.replace(/^\s*\d+[→\|→]\s?/gm, '')
}

// Clean file content for display (strip reminders and optionally line numbers)
function cleanFileContent(content: string, stripNumbers = true): string {
  let cleaned = stripSystemReminders(content)
  if (stripNumbers && /^\s*\d+[→\|]/m.test(cleaned)) {
    cleaned = stripLineNumbers(cleaned)
  }
  return cleaned
}

// Smart content renderer that detects and formats different content types
function SmartContent({ content, maxLines = 20, language }: { content: string; maxLines?: number; language?: string }) {
  const [expanded, setExpanded] = React.useState(false)
  const lines = content.split('\n')
  const needsTruncation = lines.length > maxLines
  const displayContent = expanded ? content : lines.slice(0, maxLines).join('\n')

  // Detect content type for better rendering
  const isFilePath = /^\/[\w\-./]+$/.test(content.trim())
  const isCommand = content.trim().startsWith('$') || /^(npm|git|cd|ls|cat|make|go|python|node)\s/.test(content.trim())
  const hasCodeFence = content.includes('```')

  if (isFilePath) {
    return <span className="smart-filepath">{content}</span>
  }

  if (isCommand) {
    return <code className="smart-command">{content}</code>
  }

  if (hasCodeFence) {
    // Parse markdown code blocks
    const parts = content.split(/(```\w*\n[\s\S]*?\n```)/g)
    return (
      <div className="smart-markdown">
        {parts.map((part, i) => {
          const codeMatch = part.match(/```(\w*)\n([\s\S]*?)\n```/)
          if (codeMatch) {
            const [, lang, code] = codeMatch
            return <CodeBlock key={i} code={code} language={lang || 'text'} />
          }
          return part ? <span key={i} className="smart-text">{part}</span> : null
        })}
      </div>
    )
  }

  // If a language is specified, render as code block
  if (language && language !== 'text') {
    return (
      <div className="smart-content">
        <CodeBlock code={displayContent} language={language} />
        {needsTruncation && (
          <button className="expand-btn" onClick={() => setExpanded(!expanded)}>
            {expanded ? '↑ Show less' : `↓ Show ${lines.length - maxLines} more lines`}
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="smart-content">
      <span className="smart-text">{displayContent}</span>
      {needsTruncation && (
        <>
          {!expanded && <span className="truncation-indicator">...</span>}
          <button className="expand-btn" onClick={() => setExpanded(!expanded)}>
            {expanded ? '↑ Show less' : `↓ Show ${lines.length - maxLines} more lines`}
          </button>
        </>
      )}
    </div>
  )
}

// Render tool input in a more readable format
function ToolInputDisplay({ input }: { input: any }) {
  if (!input || typeof input !== 'object') {
    return <SmartContent content={String(input || '')} />
  }

  // Special handling for common tool inputs
  const { command, file_path, content, pattern, query, code, old_string, new_string, ...rest } = input

  // Get language from file path for syntax highlighting
  const language = file_path ? getLanguageFromPath(file_path) : 'text'

  return (
    <div className="tool-input-smart">
      {file_path && (
        <div className="input-field">
          <span className="field-label">File</span>
          <span className="smart-filepath">{file_path}</span>
        </div>
      )}
      {command && (
        <div className="input-field">
          <span className="field-label">Command</span>
          <code className="smart-command">{command}</code>
        </div>
      )}
      {pattern && (
        <div className="input-field">
          <span className="field-label">Pattern</span>
          <code className="smart-pattern">{pattern}</code>
        </div>
      )}
      {query && (
        <div className="input-field">
          <span className="field-label">Query</span>
          <span className="smart-text">{query}</span>
        </div>
      )}
      {code && (
        <div className="input-field">
          <span className="field-label">Code</span>
          <CodeBlock code={code} language={language} />
        </div>
      )}
      {old_string && (
        <div className="input-field diff-field">
          <span className="field-label diff-remove">Remove</span>
          <CodeBlock code={old_string} language={language} />
        </div>
      )}
      {new_string && (
        <div className="input-field diff-field">
          <span className="field-label diff-add">Add</span>
          <CodeBlock code={new_string} language={language} />
        </div>
      )}
      {content && !code && (
        <div className="input-field">
          <span className="field-label">Content</span>
          <SmartContent
            content={typeof content === 'string' ? content : JSON.stringify(content, null, 2)}
            language={language}
            maxLines={30}
          />
        </div>
      )}
      {Object.keys(rest).length > 0 && (
        <details className="extra-fields">
          <summary>Other parameters</summary>
          <CodeBlock code={JSON.stringify(rest, null, 2)} language="json" />
        </details>
      )}
    </div>
  )
}

// Render tool output intelligently
function ToolOutputDisplay({ output, isError, filePath }: { output: any; isError: boolean; filePath?: string }) {
  const rawContent = typeof output === 'string' ? output : JSON.stringify(output, null, 2)

  // Get language from file path for file content
  const language = filePath ? getLanguageFromPath(filePath) : 'text'

  // Detect output patterns
  const hasLineNumbers = /^\s*\d+[→\|]/m.test(rawContent)
  const isFileList = /^[\w\-./]+\n[\w\-./]+/m.test(rawContent) && !hasLineNumbers

  // Clean content - strip system reminders and optionally line numbers for file content
  const isFileContent = hasLineNumbers || (filePath && !isFileList && language !== 'text')
  const content = isFileContent ? cleanFileContent(rawContent, hasLineNumbers) : stripSystemReminders(rawContent)

  const isJson = content.trim().startsWith('{') || content.trim().startsWith('[')
  const hasError = isError || /error|failed|exception/i.test(content)

  const lines = content.split('\n')
  const isLarge = lines.length > 50
  const [expanded, setExpanded] = React.useState(!isLarge)
  const displayContent = expanded ? content : lines.slice(0, 30).join('\n')

  return (
    <div className={`tool-output-smart ${hasError ? 'has-error' : ''}`}>
      {isFileContent && language !== 'text' ? (
        <>
          <CodeBlock code={displayContent} language={language} />
          {isLarge && (
            <button className="expand-btn" onClick={() => setExpanded(!expanded)}>
              {expanded ? '↑ Show less' : `↓ Show ${lines.length - 30} more lines`}
            </button>
          )}
        </>
      ) : isJson ? (
        <>
          <CodeBlock code={displayContent} language="json" />
          {isLarge && (
            <button className="expand-btn" onClick={() => setExpanded(!expanded)}>
              {expanded ? '↑ Show less' : `↓ Show ${lines.length - 30} more lines`}
            </button>
          )}
        </>
      ) : isFileList ? (
        <div className="file-list">
          {content.split('\n').filter(Boolean).slice(0, 20).map((line, i) => (
            <div key={i} className="file-list-item">{line}</div>
          ))}
          {content.split('\n').filter(Boolean).length > 20 && (
            <div className="file-list-more">+{content.split('\n').filter(Boolean).length - 20} more files</div>
          )}
        </div>
      ) : (
        <SmartContent content={content} maxLines={30} />
      )}
    </div>
  )
}

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
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M19 12H5M12 19l-7-7 7-7" />
          </svg>
          Back to sessions
        </Link>
        <div className="session-id">{session.session_id}</div>
        <div className="session-actions">
          <button className="primary-btn" onClick={() => setShareOpen(true)}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8" />
              <polyline points="16,6 12,2 8,6" />
              <line x1="12" y1="2" x2="12" y2="15" />
            </svg>
            Share
          </button>
        </div>
        <div className="session-meta-line">
          <span className="tool-badge" data-tool={getToolId(session.tool)}>
            {formatToolName(session.tool)}
          </span>
          <span>{formatTimestamp(session.created_at)}</span>
        </div>
        {session.cwd && <div className="session-meta-line">{session.cwd}</div>}
        <div className="session-meta-line">
          {messageCount} messages · {toolCount} tool calls
        </div>
      </header>

      <div className="detail-list">
        {items.map((item, idx) => {
          if (item.type === 'message') {
            const isUser = item.role === 'user'
            return (
              <article
                className={`message-card ${isUser ? 'message-user' : 'message-assistant'}`}
                key={idx}
                style={{ animationDelay: `${Math.min(idx * 30, 300)}ms` }}
              >
                <div className="message-header">
                  <div className="role">
                    {isUser ? (
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                        <circle cx="12" cy="7" r="4" />
                      </svg>
                    ) : (
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M12 2L2 7l10 5 10-5-10-5z" />
                        <path d="M2 17l10 5 10-5" />
                        <path d="M2 12l10 5 10-5" />
                      </svg>
                    )}
                    {isUser ? 'You' : 'Claude'}
                  </div>
                  {item.timestamp && <time>{formatTimestamp(item.timestamp)}</time>}
                </div>
                {item.content && (
                  <div className="message-content">
                    <SmartContent content={stripSystemReminders(item.content)} maxLines={50} />
                  </div>
                )}
                {item.thinking?.length ? (
                  <details className="thinking-block">
                    <summary>
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <circle cx="12" cy="12" r="10" />
                        <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                        <path d="M12 17h.01" />
                      </svg>
                      Thinking ({item.thinking.length} block{item.thinking.length > 1 ? 's' : ''})
                    </summary>
                    <div className="thinking-content">
                      <SmartContent content={item.thinking.join('\n\n---\n\n')} maxLines={30} />
                    </div>
                  </details>
                ) : null}
              </article>
            )
          }
          return (
            <article
              className={`tool-card${item.is_error ? ' error' : ''}`}
              key={idx}
              style={{ animationDelay: `${Math.min(idx * 30, 300)}ms` }}
            >
              <div className="tool-title">
                <span className="tool-name">
                  <ToolIcon name={item.tool_name} />
                  {item.tool_name}
                </span>
                <time>{formatTimestamp(item.timestamp)}</time>
              </div>
              {item.input && (
                <details open={!isLargeInput(item.input)}>
                  <summary>Input</summary>
                  <ToolInputDisplay input={item.input} />
                </details>
              )}
              {item.output && (
                <details open={!isLargeOutput(item.output)}>
                  <summary>Output {item.is_error && <span className="error-badge">Error</span>}</summary>
                  <ToolOutputDisplay output={item.output} isError={item.is_error} filePath={item.input?.file_path} />
                </details>
              )}
            </article>
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
      items.push({ type: 'message', role: data.role, content: parts.text, thinking: parts.thinking, timestamp: event.timestamp })
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

function formatToolName(tool: string) {
  const t = tool?.toLowerCase().replace(/-/g, '_') || ''
  if (t === 'claude_code') return 'Claude'
  if (t === 'cursor') return 'Cursor'
  return tool?.replace(/[-_]/g, ' ').replace(/\b\w/g, c => c.toUpperCase()) || ''
}

// Tool-specific icons
function ToolIcon({ name }: { name: string }) {
  const n = name?.toLowerCase() || ''

  if (n.includes('read') || n.includes('file')) {
    return (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
        <path d="M14 2v6h6" />
        <path d="M16 13H8M16 17H8M10 9H8" />
      </svg>
    )
  }
  if (n.includes('write') || n.includes('edit')) {
    return (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
      </svg>
    )
  }
  if (n.includes('bash') || n.includes('command') || n.includes('shell')) {
    return (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <polyline points="4 17 10 11 4 5" />
        <line x1="12" y1="19" x2="20" y2="19" />
      </svg>
    )
  }
  if (n.includes('glob') || n.includes('grep') || n.includes('search')) {
    return (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <circle cx="11" cy="11" r="8" />
        <path d="M21 21l-4.35-4.35" />
      </svg>
    )
  }
  if (n.includes('web') || n.includes('fetch') || n.includes('url')) {
    return (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
      </svg>
    )
  }
  if (n.includes('task') || n.includes('agent')) {
    return (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
        <circle cx="8.5" cy="7" r="4" />
        <path d="M20 8v6M23 11h-6" />
      </svg>
    )
  }
  // Default tool icon
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
    </svg>
  )
}

function isLargeInput(input: any): boolean {
  if (!input) return false
  const str = typeof input === 'string' ? input : JSON.stringify(input)
  return str.length > 500
}

function isLargeOutput(output: any): boolean {
  if (!output) return false
  const str = typeof output === 'string' ? output : JSON.stringify(output)
  return str.length > 1000
}

function getToolId(tool: string) {
  return tool?.toLowerCase().replace(/[-\s]+/g, '_') || ''
}

function formatTimestamp(value: string | undefined) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}
