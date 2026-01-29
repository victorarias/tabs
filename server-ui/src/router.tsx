import { createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen'
import { DefaultCatchBoundary } from './components/DefaultCatchBoundary'
import { NotFound } from './components/NotFound'

function parseSearch(searchStr: string) {
  const raw = searchStr.startsWith('?') ? searchStr.slice(1) : searchStr
  const params = new URLSearchParams(raw)
  const result: Record<string, unknown> = {}
  for (const [key, value] of params.entries()) {
    const existing = result[key]
    if (existing === undefined) {
      result[key] = value
    } else if (Array.isArray(existing)) {
      existing.push(value)
    } else {
      result[key] = [existing, value]
    }
  }
  return result
}

function stringifySearch(search: Record<string, unknown>) {
  const params = new URLSearchParams()
  Object.entries(search).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    if (Array.isArray(value)) {
      value.forEach((item) => {
        if (item === undefined || item === null || item === '') return
        params.append(key, String(item))
      })
      return
    }
    params.set(key, String(value))
  })
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function getRouter() {
  const router = createRouter({
    routeTree,
    defaultPreload: 'intent',
    defaultErrorComponent: DefaultCatchBoundary,
    defaultNotFoundComponent: () => <NotFound />,
    scrollRestoration: true,
    parseSearch,
    stringifySearch,
  })
  return router
}
