import { createFileRoute } from '@tanstack/react-router'
import { pushSession } from '~/utils/push'

export const Route = createFileRoute('/api/sessions/push')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const payload = await request.json().catch(() => null)
        if (!payload || typeof payload !== 'object') {
          return Response.json(
            { status: 'error', error: { code: 'invalid_request', message: 'Invalid JSON body' } },
            { status: 400 },
          )
        }
        const sessionId = String(payload.session_id || '').trim()
        const tool = String(payload.tool || '').trim()
        const tags = Array.isArray(payload.tags) ? payload.tags : []
        if (!sessionId || !tool) {
          return Response.json(
            { status: 'error', error: { code: 'invalid_request', message: 'Missing session_id or tool' } },
            { status: 400 },
          )
        }
        try {
          const result = await pushSession(sessionId, tool, tags)
          return Response.json({ status: 'ok', remote_id: result.id, url: result.url })
        } catch (err: any) {
          return Response.json(
            { status: 'error', error: { code: 'push_failed', message: err?.message || 'Push failed' } },
            { status: 400 },
          )
        }
      },
    },
  },
})
