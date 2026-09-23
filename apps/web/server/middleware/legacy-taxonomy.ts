import {
  parseWikiId,
  resolveLegacyTag,
  resolveLegacyEngine,
  resolveRenamedTaxonomyPath
} from '../utils/kunTaxonomyRedirects'
import { taxonomyDetailPath } from '../../shared/utils/kunTaxonomyPaths'
import { createApiClient } from '../../shared/utils/api/client'
import { settle } from '../../shared/utils/api/problem'

const LEGACY_RE = /^\/galgame-(tag|official|engine)\/(\d+)$/

export default defineEventHandler(async (event) => {
  if (event.method !== 'GET') return
  const [path, query] = event.path.split('?') as [string, string?]

  const withQuery = (to: string) => (query ? `${to}?${query}` : to)

  const renamed = resolveRenamedTaxonomyPath(path)
  if (renamed) return sendRedirect(event, withQuery(renamed), 301)

  const m = path.match(LEGACY_RE)
  if (!m) return

  const family = m[1]!
  const wikiId = parseWikiId(m[2])
  if (wikiId === null) return

  if (family === 'tag') {
    const res = resolveLegacyTag(wikiId)
    if (res.kind === 'redirect') return sendRedirect(event, res.to, 301)
    if (res.kind === 'gone') {
      throw createError({
        statusCode: 410,
        statusMessage: '该标签已在词表迁移中退役'
      })
    }
    return
  }

  if (family === 'engine') {
    const res = resolveLegacyEngine(wikiId)
    if (res.kind === 'redirect') return sendRedirect(event, res.to, 301)
    return
  }

  const res = await settle(
    createApiClient({
      origin: useRuntimeConfig().apiBaseUrl,
      timeoutMs: 5000
    }).GET('/wiki-company-redirects/{wiki_company_id}', {
      params: { path: { wiki_company_id: String(wikiId) } }
    })
  )

  if (res.ok) {
    return sendRedirect(
      event,
      taxonomyDetailPath('official', Number(res.data.company_id)),
      301
    )
  }
})
