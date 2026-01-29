import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'

export type Config = {
  local: {
    ui_port: number
    log_level: string
  }
  remote: {
    server_url: string
    api_key: string
    auto_push: boolean
    default_tags: string[]
  }
  cursor: {
    db_path: string
    poll_interval: number
  }
  claude_code: {
    projects_dir: string
  }
}

export function defaultConfig(): Config {
  return {
    local: { ui_port: 3787, log_level: 'info' },
    remote: {
      server_url: 'https://tabs.company.com',
      api_key: '',
      auto_push: false,
      default_tags: [],
    },
    cursor: { db_path: '', poll_interval: 2 },
    claude_code: { projects_dir: '' },
  }
}

export function configPath(): string {
  return path.join(os.homedir(), '.tabs', 'config.toml')
}

export async function loadConfig(): Promise<Config> {
  try {
    const data = await fs.readFile(configPath(), 'utf8')
    return parseConfig(data)
  } catch {
    return defaultConfig()
  }
}

export async function writeConfig(cfg: Config): Promise<void> {
  const dir = path.dirname(configPath())
  await fs.mkdir(dir, { recursive: true, mode: 0o700 })
  await fs.writeFile(configPath(), formatConfig(cfg), { mode: 0o600 })
}

export function parseConfig(raw: string): Config {
  const cfg = defaultConfig()
  let section = ''
  const lines = raw.split(/\r?\n/)
  for (const line of lines) {
    const cleaned = stripComment(line.trim())
    if (!cleaned) continue
    if (cleaned.startsWith('[') && cleaned.endsWith(']')) {
      section = cleaned.slice(1, -1).trim()
      continue
    }
    const parts = cleaned.split('=')
    if (parts.length < 2) continue
    const key = parts.shift()?.trim() ?? ''
    const value = parts.join('=').trim()
    applyValue(cfg, section, key, value)
  }
  return cfg
}

function stripComment(line: string): string {
  if (!line.includes('#')) return line
  let out = ''
  let inQuotes = false
  for (const ch of line) {
    if (ch === '"') inQuotes = !inQuotes
    if (ch === '#' && !inQuotes) break
    out += ch
  }
  return out.trim()
}

function parseTomlValue(raw: string): unknown {
  const trimmed = raw.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    const inner = trimmed.slice(1, -1).trim()
    if (!inner) return []
    return inner
      .split(',')
      .map((part) => part.trim())
      .filter(Boolean)
      .map((part) => part.replace(/^"|"$/g, ''))
  }
  if (trimmed === 'true' || trimmed === 'false') {
    return trimmed === 'true'
  }
  if (/^-?\d+$/.test(trimmed)) {
    return Number(trimmed)
  }
  return trimmed.replace(/^"|"$/g, '')
}

function applyValue(cfg: Config, section: string, key: string, raw: string) {
  const value = parseTomlValue(raw)
  switch (section) {
    case 'local':
      if (key === 'ui_port' && typeof value === 'number') cfg.local.ui_port = value
      if (key === 'log_level' && typeof value === 'string') cfg.local.log_level = value
      break
    case 'remote':
      if (key === 'server_url' && typeof value === 'string') cfg.remote.server_url = value
      if (key === 'api_key' && typeof value === 'string') cfg.remote.api_key = value
      if (key === 'auto_push' && typeof value === 'boolean') cfg.remote.auto_push = value
      if (key === 'default_tags' && Array.isArray(value)) cfg.remote.default_tags = value
      break
    case 'cursor':
      if (key === 'db_path' && typeof value === 'string') cfg.cursor.db_path = value
      if (key === 'poll_interval' && typeof value === 'number') cfg.cursor.poll_interval = value
      break
    case 'claude_code':
      if (key === 'projects_dir' && typeof value === 'string') cfg.claude_code.projects_dir = value
      break
  }
}

export function applySet(cfg: Config, key: string, rawValue: string): void {
  const normalized = normalizeKey(key)
  switch (normalized) {
    case 'remote.server_url':
    case 'server_url':
      cfg.remote.server_url = rawValue.trim()
      return
    case 'remote.api_key':
    case 'api_key':
      cfg.remote.api_key = rawValue.trim()
      return
    case 'remote.auto_push':
    case 'auto_push':
      cfg.remote.auto_push = rawValue.trim() === 'true'
      return
    case 'remote.default_tags':
    case 'default_tags':
      cfg.remote.default_tags = parseTags(rawValue)
      return
    case 'local.ui_port':
    case 'ui_port':
      cfg.local.ui_port = Number(rawValue)
      return
    case 'local.log_level':
    case 'log_level':
      cfg.local.log_level = rawValue.trim()
      return
    default:
      return
  }
}

function normalizeKey(key: string): string {
  return key.trim().toLowerCase().replace(/-/g, '_')
}

function parseTags(raw: string): string[] {
  const trimmed = raw.trim()
  if (!trimmed) return []
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    try {
      const parsed = JSON.parse(trimmed)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return trimmed
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
}

function formatStringArray(values: string[]): string {
  if (!values.length) return '[]'
  return `[${values.map((value) => `"${value}"`).join(', ')}]`
}

export function formatConfig(cfg: Config): string {
  return [
    '# tabs configuration file',
    '# Generated by: tabs-ui',
    '',
    '[local]',
    `ui_port = ${cfg.local.ui_port}`,
    `log_level = "${cfg.local.log_level}"`,
    '',
    '[remote]',
    `server_url = "${cfg.remote.server_url}"`,
    `api_key = "${cfg.remote.api_key}"`,
    `auto_push = ${cfg.remote.auto_push}`,
    `default_tags = ${formatStringArray(cfg.remote.default_tags)}`,
    '',
    '[cursor]',
    `db_path = "${cfg.cursor.db_path}"`,
    `poll_interval = ${cfg.cursor.poll_interval}`,
    '',
    '[claude_code]',
    `projects_dir = "${cfg.claude_code.projects_dir}"`,
    '',
  ].join('\n')
}
