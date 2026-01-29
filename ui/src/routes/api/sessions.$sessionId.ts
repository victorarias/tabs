import { createFileRoute } from '@tanstack/react-router'
import { getSession } from '~/utils/localSessions'

export const Route = createFileRoute('/api/sessions/$sessionId')({
  server: {
    handlers: {
      GET: async ({ params }) => {
        const session = await getSession(params.sessionId)
        if (!session) {
          return new Response(
            JSON.stringify({
              status: 'error',
              error: { code: 'session_not_found', message: 'Session not found' },
            }),
            { status: 404, headers: { 'Content-Type': 'application/json' } },
          )
        }
        return Response.json({ session })
      },
    },
  },
})
