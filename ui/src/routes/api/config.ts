import { createFileRoute } from '@tanstack/react-router'
import { applySet, loadConfig, writeConfig } from '~/utils/config'

export const Route = createFileRoute('/api/config')({
  server: {
    handlers: {
      GET: async () => {
        const cfg = await loadConfig()
        const apiKeyPrefix = cfg.remote.api_key ? cfg.remote.api_key.slice(0, 12) : ''
        return Response.json({
          local: {
            ui_port: cfg.local.ui_port,
            log_level: cfg.local.log_level,
          },
          remote: {
            server_url: cfg.remote.server_url,
            api_key_configured: Boolean(cfg.remote.api_key),
            api_key_prefix: apiKeyPrefix,
            default_tags: cfg.remote.default_tags,
          },
        })
      },
      PUT: async ({ request }) => {
        const payload = await request.json().catch(() => null)
        if (!payload || typeof payload !== 'object') {
          return Response.json(
            { status: 'error', error: { code: 'invalid_request', message: 'Invalid JSON body' } },
            { status: 400 },
          )
        }
        const cfg = await loadConfig()
        if (payload.remote?.server_url !== undefined) {
          applySet(cfg, 'server_url', String(payload.remote.server_url))
        }
        if (payload.remote?.api_key !== undefined) {
          applySet(cfg, 'api_key', String(payload.remote.api_key))
        }
        if (payload.remote?.default_tags !== undefined) {
          applySet(cfg, 'default_tags', String(payload.remote.default_tags))
        }
        await writeConfig(cfg)
        return Response.json({ status: 'ok', message: 'Configuration updated' })
      },
    },
  },
})
