import { createApiClient, sessionCookie } from '../../shared/utils/api/client'
import { settle } from '../../shared/utils/api/problem'

const LEGACY_RE = /^\/galgame\/collection\/(\d+)$/

export default defineEventHandler(async (event) => {
  if (event.method !== 'GET') return
  const [path, query] = event.path.split('?') as [string, string?]

  const m = path.match(LEGACY_RE)
  if (!m) return

  const aliasId = m[1]
  if (!aliasId) return

  const res = await settle(
    createApiClient({
      origin: useRuntimeConfig().apiBaseUrl,
      cookie: sessionCookie(getHeader(event, 'cookie')),
      timeoutMs: 5000
    }).GET('/collection-aliases/{alias_id}', {
      params: { path: { alias_id: aliasId } }
    })
  )

  if (res.ok) {
    const to = `/collection/${res.data.collection_id}`
    return sendRedirect(event, query ? `${to}?${query}` : to, 301)
  }
})
