import { createFileRoute } from '@tanstack/react-router'
import { listSessions } from '~/utils/localSessions'

export const Route = createFileRoute('/api/sessions')({
  server: {
    handlers: {
      GET: async ({ request }) => {
        const url = new URL(request.url)
        const filter = {
          tool: url.searchParams.get('tool') || undefined,
          date: url.searchParams.get('date') || undefined,
          cwd: url.searchParams.get('cwd') || undefined,
          q: url.searchParams.get('q') || undefined,
        }
        const sessions = await listSessions(filter)
        return Response.json({ sessions, total: sessions.length })
      },
    },
  },
})
