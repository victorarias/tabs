import { tanstackStart } from '@tanstack/react-start/plugin/vite'
import { defineConfig } from 'vite'
import httpProxy from 'http-proxy'
import tsConfigPaths from 'vite-tsconfig-paths'
import viteReact from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { nitro } from 'nitro/vite'

function apiProxy(target: string) {
  const { createProxyServer } = httpProxy as unknown as {
    createProxyServer: (opts: { target: string; changeOrigin: boolean }) => any
  }
  const proxy = createProxyServer({ target, changeOrigin: true })
  proxy.on('error', (_err, _req, res) => {
    if (!res || res.headersSent) return
    res.writeHead(502, { 'Content-Type': 'text/plain' })
    res.end('Bad gateway')
  })
  return {
    name: 'api-proxy',
    configureServer(server: any) {
      server.middlewares.use((req: any, res: any, next: any) => {
        if (req?.url && req.url.startsWith('/api')) {
          proxy.web(req, res, {}, (err: any) => next(err))
          return
        }
        next()
      })
    },
  }
}

export default defineConfig({
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  plugins: [
    apiProxy('http://127.0.0.1:8080'),
    tailwindcss(),
    tsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    tanstackStart({
      srcDirectory: 'src',
    }),
    viteReact(),
    nitro(),
  ],
})
