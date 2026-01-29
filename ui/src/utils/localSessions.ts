import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'

export type SessionSummary = {
  session_id: string
  tool: string
  created_at: string
  ended_at?: string
  cwd?: string
  summary?: string
  duration_seconds?: number
  message_count: number
  tool_use_count: number
  file_path: string
}

export type SessionDetail = {
  session_id: string
  tool: string
  created_at: string
  ended_at?: string
  cwd?: string
  duration_seconds?: number
  events: Record<string, any>[]
}

export type SessionFilter = {
  tool?: string
  date?: string
  cwd?: string
  q?: string
}

const sessionsDir = () => path.join(os.homedir(), '.tabs', 'sessions')

export async function listSessions(filter: SessionFilter): Promise<SessionSummary[]> {
  const dir = sessionsDir()
  let entries: string[] = []
  try {
    entries = await fs.readdir(dir)
  } catch {
    return []
  }

  const summaries: SessionSummary[] = []
  for (const entry of entries) {
    const dayDir = path.join(dir, entry)
    let files: string[] = []
    try {
      const stat = await fs.stat(dayDir)
      if (!stat.isDirectory()) continue
      files = await fs.readdir(dayDir)
    } catch {
      continue
    }

    for (const file of files) {
      if (!file.endsWith('.jsonl')) continue
      const filePath = path.join(dayDir, file)
      const { summary, matched } = await summarizeSession(filePath, filter)
      if (!matched) continue
      if (filter.date) {
        if (summary.created_at) {
          const created = summary.created_at.slice(0, 10)
          if (created !== filter.date) continue
        } else if (entry !== filter.date) {
          continue
        }
      }
      if (filter.tool && summary.tool !== filter.tool) continue
      if (filter.cwd && summary.cwd && !summary.cwd.startsWith(filter.cwd)) continue
      summaries.push(summary)
    }
  }

  summaries.sort((a, b) => (b.created_at || '').localeCompare(a.created_at || ''))
  return summaries
}

export async function getSession(sessionId: string, tool?: string): Promise<SessionDetail | null> {
  const filePath = await findSessionFile(sessionId, tool)
  if (!filePath) return null
  return loadSessionDetail(filePath)
}

async function findSessionFile(sessionId: string, tool?: string): Promise<string | null> {
  const dir = sessionsDir()
  let entries: string[] = []
  try {
    entries = await fs.readdir(dir)
  } catch {
    return null
  }

  let bestPath: string | null = null
  let bestTs = -1
  for (const entry of entries) {
    const dayDir = path.join(dir, entry)
    let files: string[] = []
    try {
      const stat = await fs.stat(dayDir)
      if (!stat.isDirectory()) continue
      files = await fs.readdir(dayDir)
    } catch {
      continue
    }

    for (const file of files) {
      if (!file.endsWith('.jsonl')) continue
      if (!file.startsWith(sessionId + '-')) continue
      if (tool && !file.startsWith(`${sessionId}-${tool}-`)) continue
      const ts = extractTimestamp(file)
      if (ts > bestTs) {
        bestTs = ts
        bestPath = path.join(dayDir, file)
      }
    }
  }
  return bestPath
}

function extractTimestamp(filename: string): number {
  const trimmed = filename.replace(/\.jsonl$/, '')
  const parts = trimmed.split('-')
  if (parts.length < 3) return -1
  const ts = Number(parts[parts.length - 1])
  return Number.isFinite(ts) ? ts : -1
}

async function summarizeSession(filePath: string, filter: SessionFilter) {
  const data = await fs.readFile(filePath, 'utf8')
  const lines = data.split(/\r?\n/)
  const summary: SessionSummary = {
    session_id: '',
    tool: '',
    created_at: '',
    cwd: '',
    summary: '',
    duration_seconds: 0,
    message_count: 0,
    tool_use_count: 0,
    file_path: filePath,
  }

  let earliest = ''
  let latest = ''
  let hasStart = false
  let overrideCounts = false
  let matchedQuery = !filter.q
  let firstUserSummary = ''

  for (const line of lines) {
    if (!line.trim()) continue
    let event: any
    try {
      event = JSON.parse(line)
    } catch {
      continue
    }

    if (!summary.session_id && event.session_id) summary.session_id = event.session_id
    if (!summary.tool && event.tool) summary.tool = event.tool

    const ts = event.timestamp || ''
    if (ts) {
      if (!earliest || ts < earliest) earliest = ts
      if (!latest || ts > latest) latest = ts
    }

    const eventType = event.event_type
    if (eventType === 'session_start') {
      if (!hasStart && ts) {
        summary.created_at = ts
        hasStart = true
      }
      if (event.data?.cwd) summary.cwd = event.data.cwd
    }

    if (eventType === 'session_end') {
      summary.ended_at = ts
      if (event.data?.duration_seconds) summary.duration_seconds = event.data.duration_seconds
      if (event.data?.message_count) {
        summary.message_count = event.data.message_count
        overrideCounts = true
      }
      if (event.data?.tool_use_count) {
        summary.tool_use_count = event.data.tool_use_count
        overrideCounts = true
      }
    }

    if (eventType === 'message') {
      if (!overrideCounts) summary.message_count += 1
      if (!firstUserSummary && event.data?.role === 'user') {
        const text = extractMessageSummary(event.data)
        if (text) firstUserSummary = text
      }
      if (!matchedQuery && matchesQuery(event, filter.q || '')) matchedQuery = true
    }

    if (eventType === 'tool_use') {
      if (!overrideCounts) summary.tool_use_count += 1
    }

    if (!matchedQuery && matchesQuery(event, filter.q || '')) matchedQuery = true
  }

  if (!summary.created_at && earliest) summary.created_at = earliest
  if (!summary.ended_at && latest) summary.ended_at = latest
  if (!summary.duration_seconds && earliest && latest) {
    const duration = Math.max(0, Date.parse(latest) - Date.parse(earliest))
    summary.duration_seconds = Math.floor(duration / 1000)
  }
  if (!summary.summary) {
    summary.summary = (firstUserSummary || '').slice(0, 160)
  }

  return { summary, matched: matchedQuery }
}

async function loadSessionDetail(filePath: string): Promise<SessionDetail> {
  const data = await fs.readFile(filePath, 'utf8')
  const lines = data.split(/\r?\n/)
  const detail: SessionDetail = {
    session_id: '',
    tool: '',
    created_at: '',
    cwd: '',
    duration_seconds: 0,
    events: [],
  }
  let earliest = ''
  let latest = ''
  let durationSeconds = 0

  for (const line of lines) {
    if (!line.trim()) continue
    let event: any
    try {
      event = JSON.parse(line)
    } catch {
      continue
    }

    if (!detail.session_id && event.session_id) detail.session_id = event.session_id
    if (!detail.tool && event.tool) detail.tool = event.tool

    const ts = event.timestamp || ''
    if (ts) {
      if (!earliest || ts < earliest) earliest = ts
      if (!latest || ts > latest) latest = ts
    }

    if (event.event_type === 'session_start') {
      if (event.data?.cwd) detail.cwd = event.data.cwd
      if (ts) detail.created_at = ts
    }
    if (event.event_type === 'session_end') {
      detail.ended_at = ts
      if (event.data?.duration_seconds) durationSeconds = event.data.duration_seconds
    }

    detail.events.push(event)
  }

  if (!detail.created_at && earliest) detail.created_at = earliest
  if (!detail.ended_at && latest) detail.ended_at = latest
  if (!durationSeconds && earliest && latest) {
    const duration = Math.max(0, Date.parse(latest) - Date.parse(earliest))
    durationSeconds = Math.floor(duration / 1000)
  }
  if (durationSeconds) detail.duration_seconds = durationSeconds

  return detail
}

function matchesQuery(event: any, query: string): boolean {
  if (!query) return true
  const haystack = JSON.stringify(event).toLowerCase()
  return haystack.includes(query.toLowerCase())
}

function extractMessageSummary(data: any): string {
  if (!data) return ''
  if (Array.isArray(data.content)) {
    const text = data.content
      .map((part: { text?: string } | string) =>
        typeof part === 'string' ? part : part?.text || '',
      )
      .filter(Boolean)
      .join('\n')
    return text.trim()
  }
  if (typeof data.content === 'string') return data.content.trim()
  return ''
}
