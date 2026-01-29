import { loadConfig } from './config'
import { getSession } from './localSessions'

type PushTag = { key: string; value: string }

export async function pushSession(sessionId: string, tool: string, tags: PushTag[]) {
  const cfg = await loadConfig()
  if (!cfg.remote.server_url) {
    throw new Error('server_url not configured')
  }
  if (!cfg.remote.api_key) {
    throw new Error('api_key not configured')
  }

  const session = await getSession(sessionId, tool)
  if (!session) {
    throw new Error('session not found')
  }

  const payload = {
    session: {
      session_id: session.session_id,
      tool: session.tool,
      created_at: session.created_at,
      ended_at: session.ended_at ?? '',
      cwd: session.cwd ?? '',
      events: session.events,
    },
    tags: mergeTags(cfg.remote.default_tags, tags),
  }

  const resp = await fetch(`${cfg.remote.server_url.replace(/\/$/, '')}/api/sessions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${cfg.remote.api_key}`,
    },
    body: JSON.stringify(payload),
  })

  if (!resp.ok) {
    const data = await resp.json().catch(() => null)
    const message = data?.error?.message || 'push failed'
    throw new Error(message)
  }

  return resp.json()
}

function mergeTags(defaults: string[], tags: PushTag[]): PushTag[] {
  const seen = new Set<string>()
  const out: PushTag[] = []
  for (const entry of defaults || []) {
    const parsed = parseTag(entry)
    if (!parsed) continue
    const key = `${parsed.key}:${parsed.value}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(parsed)
  }
  for (const tag of tags || []) {
    const key = `${tag.key}:${tag.value}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(tag)
  }
  return out
}

function parseTag(raw: string): PushTag | null {
  if (!raw) return null
  const parts = raw.includes(':') ? raw.split(':') : raw.split('=')
  if (parts.length < 2) return null
  const key = parts[0].trim()
  const value = parts.slice(1).join(':').trim()
  if (!key || !value) return null
  return { key, value }
}
