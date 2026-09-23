import { createApiClient, type ApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'

interface SitemapUrl {
  loc: string
  lastmod?: string
  changefreq?: string
  priority?: number
}

const TOPIC_PAGE_LIMIT = 100
const TOPIC_MAX_PAGES = 200

export const collectTopicUrls = async (
  api: ApiClient
): Promise<SitemapUrl[]> => {
  // GET /api/topic started returning { topics, total } on 2026-06-24, so
  // pick: Array.isArray was [] and the sitemap carried no topic URLs.
  const urls: SitemapUrl[] = []
  let cursor: string | undefined
  for (let page = 0; page < TOPIC_MAX_PAGES; page++) {
    const result = await settle(
      api.GET('/topics', {
        params: {
          query: {
            limit: TOPIC_PAGE_LIMIT,
            ...(cursor ? { cursor } : {})
          }
        }
      })
    )
    if (!result.ok) {
      return urls
    }
    for (const item of result.data.items) {
      urls.push({
        loc: `/topic/${item.id}`,
        lastmod: item.bumped_at,
        changefreq: 'daily',
        priority: 0.8
      })
    }
    if (!result.data.next_cursor) {
      break
    }
    cursor = result.data.next_cursor
  }
  return urls
}

const ENTITY_PAGE_LIMIT = 100
const ENTITY_MAX_PAGES = 100

export const collectRatingUrls = async (
  api: ApiClient
): Promise<SitemapUrl[]> => {
  const urls: SitemapUrl[] = []
  for (let page = 1; page <= ENTITY_MAX_PAGES; page++) {
    const result = await settle(
      api.GET('/ratings', {
        params: { query: { page, limit: ENTITY_PAGE_LIMIT } }
      })
    )
    if (!result.ok) {
      break
    }
    for (const item of result.data.items) {
      urls.push({
        loc: `/galgame-rating/${item.id}`,
        lastmod: item.updated_at,
        changefreq: 'daily',
        priority: 0.6
      })
    }
    if (page * ENTITY_PAGE_LIMIT >= result.data.total) {
      break
    }
  }
  return urls
}

type EntityPage = { items: { id: string }[]; total: number }

const collectPagedIds = async (
  fetchPage: (page: number) => Promise<EntityPage | null>,
  loc: (id: string) => string
): Promise<SitemapUrl[]> => {
  const urls: SitemapUrl[] = []
  for (let page = 1; page <= ENTITY_MAX_PAGES; page++) {
    const data = await fetchPage(page)
    if (!data) {
      break
    }
    for (const item of data.items) {
      urls.push({ loc: loc(item.id), changefreq: 'daily', priority: 0.5 })
    }
    if (page * ENTITY_PAGE_LIMIT >= data.total) {
      break
    }
  }
  return urls
}

// Anonymous reads, so the adult tags stay out as they do for any SFW reader.
export const collectEntityUrls = async (
  api: ApiClient
): Promise<SitemapUrl[]> => {
  const query = (page: number) => ({ page, limit: ENTITY_PAGE_LIMIT })
  const ok = <T>(r: { ok: true; data: T } | { ok: false }) =>
    r.ok ? r.data : null
  const groups = await Promise.all([
    collectPagedIds(
      async (page) =>
        ok(await settle(api.GET('/companies', { params: { query: query(page) } }))),
      (id) => `/galgame/official/${id}`
    ),
    collectPagedIds(
      async (page) =>
        ok(await settle(api.GET('/tags', { params: { query: query(page) } }))),
      (id) => `/galgame/tag/${id}`
    ),
    collectPagedIds(
      async (page) =>
        ok(await settle(api.GET('/engines', { params: { query: query(page) } }))),
      (id) => `/galgame/engine/${id}`
    )
  ])
  return groups.flat()
}

const SFW_COOKIE = `KUNGalgameSettings=${encodeURIComponent(
  JSON.stringify({ showKUNGalgameContentLimit: 'sfw' })
)}`

const PAGE_SIZE = 50
const MAX_PAGES = 130
const WORK_PAGE_LIMIT = 50
const WORK_MAX_PAGES = 200
const GLOBAL_CONCURRENCY = 12

const createLimiter = (max: number) => {
  let active = 0
  const waiters: Array<() => void> = []
  const release = () => {
    active--
    waiters.shift()?.()
  }
  return async <T>(fn: () => Promise<T>): Promise<T> => {
    if (active >= max)
      await new Promise<void>((resolve) => waiters.push(resolve))
    active++
    try {
      return await fn()
    } finally {
      release()
    }
  }
}

const range = (from: number, to: number): number[] =>
  Array.from({ length: Math.max(0, to - from + 1) }, (_, i) => from + i)

export const collectWorkUrls = async (
  api: ApiClient
): Promise<SitemapUrl[]> => {
  const limit = createLimiter(GLOBAL_CONCURRENCY)
  const query = (page: number) => ({
    page,
    limit: WORK_PAGE_LIMIT,
    sort: 'resource_updated_desc' as const,
    include_resourceless: true
  })
  const fetchPage = (page: number) =>
    limit(async () => {
      const result = await settle(
        api.GET('/works', { params: { query: query(page) } })
      )
      return result.ok ? result.data : null
    })

  const urls: SitemapUrl[] = []
  const push = (
    items: { id: string; resource_updated_at: string | null }[]
  ) => {
    for (const item of items) {
      urls.push({
        loc: `/galgame/${item.id}`,
        ...(item.resource_updated_at
          ? { lastmod: item.resource_updated_at }
          : {}),
        changefreq: 'daily',
        priority: 0.8
      })
    }
  }

  const first = await fetchPage(1)
  if (!first) {
    return urls
  }
  push(first.items)
  const pages = Math.min(
    Math.ceil(first.total / WORK_PAGE_LIMIT),
    WORK_MAX_PAGES
  )
  const rest = await Promise.all(
    range(2, pages).map((page) => fetchPage(page))
  )
  for (const body of rest) {
    if (body) {
      push(body.items)
    }
  }
  return urls
}

const unwrap = (json: unknown): unknown =>
  (json as { data?: unknown })?.data ?? json

const toIso = (d: unknown): string | undefined => {
  if (typeof d === 'string' || typeof d === 'number' || d instanceof Date) {
    const t = new Date(d)
    if (!Number.isNaN(t.getTime())) return t.toISOString()
  }
  return undefined
}

interface PagedSource {
  path: string
  pick: (data: unknown) => Record<string, unknown>[]
  total?: (data: unknown) => number | undefined
  loc: (row: Record<string, unknown>) => string
  lastmod?: (row: Record<string, unknown>) => string | undefined
  priority: number
}

export const buildSitemapUrls = async (
  apiBase: string
): Promise<SitemapUrl[]> => {
  const limit = createLimiter(GLOBAL_CONCURRENCY)

  const apiGet = (path: string): Promise<unknown | null> =>
    limit(async () => {
      try {
        return await $fetch(`${apiBase}/api${path}`, {
          headers: { cookie: SFW_COOKIE, accept: 'application/json' },
          timeout: 15000
        })
      } catch {
        return null
      }
    })

  const withPage = (path: string, page: number) => {
    const sep = path.includes('?') ? '&' : '?'
    return `${path}${sep}page=${page}&limit=${PAGE_SIZE}`
  }

  const fetchPage = async (src: PagedSource, page: number) => {
    const body = await apiGet(withPage(src.path, page))
    return body ? src.pick(unwrap(body)) : []
  }

  const toUrls = (
    src: PagedSource,
    rows: Record<string, unknown>[]
  ): SitemapUrl[] =>
    rows.map((row) => ({
      loc: src.loc(row),
      lastmod: src.lastmod?.(row),
      changefreq: 'daily',
      priority: src.priority
    }))

  const collect = async (src: PagedSource): Promise<SitemapUrl[]> => {
    const firstBody = await apiGet(withPage(src.path, 1))
    if (!firstBody) return []
    const firstRows = src.pick(unwrap(firstBody))
    const urls = toUrls(src, firstRows)

    if (src.total) {
      const total = src.total(unwrap(firstBody))
      const pages =
        typeof total === 'number' && total > 0
          ? Math.min(Math.ceil(total / PAGE_SIZE), MAX_PAGES)
          : MAX_PAGES
      const rest = await Promise.all(
        range(2, pages).map((p) =>
          fetchPage(src, p).then((r) => toUrls(src, r))
        )
      )
      for (const r of rest) urls.push(...r)
      return urls
    }

    for (let start = 2; start <= MAX_PAGES; start += GLOBAL_CONCURRENCY) {
      const batch = await Promise.all(
        range(start, Math.min(start + GLOBAL_CONCURRENCY - 1, MAX_PAGES)).map(
          (p) => fetchPage(src, p)
        )
      )
      let reachedEnd = false
      for (const rows of batch) {
        urls.push(...toUrls(src, rows))
        if (rows.length === 0) reachedEnd = true
      }
      if (reachedEnd) break
    }
    return urls
  }

  const num = (row: Record<string, unknown>, key: string) => row[key] as number

  const paged: PagedSource[] = [
    {
      path: '/galgame-resource',
      pick: (d) =>
        ((d as { resources?: [] })?.resources ?? []) as Record<
          string,
          unknown
        >[],
      total: (d) => (d as { total?: number })?.total,
      loc: (r) => `/galgame/resource/${num(r, 'id')}`,
      lastmod: (r) => toIso(r.edited ?? r.created),
      priority: 0.6
    }
  ]

  const typed = createApiClient({ origin: apiBase, timeoutMs: 15000 })
  const groups = await Promise.all([
    collectTopicUrls(typed),
    collectWorkUrls(typed),
    ...paged.map((src) => collect(src)),
    collectEntityUrls(typed),
    collectRatingUrls(typed)
  ])

  const seen = new Set<string>()
  const out: SitemapUrl[] = []
  for (const url of groups.flat()) {
    if (seen.has(url.loc)) continue
    seen.add(url.loc)
    out.push(url)
  }
  return out
}
