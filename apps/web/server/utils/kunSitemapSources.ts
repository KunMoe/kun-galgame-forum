import { createApiClient, type ApiClient } from '#shared/utils/api/client'
import {
  isTransientProblem,
  settle,
  type ApiResult
} from '#shared/utils/api/problem'

interface SitemapUrl {
  loc: string
  lastmod?: string
  changefreq?: string
  priority?: number
}

type Requester = <T>(run: () => Promise<ApiResult<T>>) => Promise<ApiResult<T>>

const direct: Requester = (run) => run()

const RETRY_DELAYS_MS = [1000, 3000, 9000]

// Every source runs at once, so without a shared ceiling the build fired a
// burst that the news service answered with 429, and each source stopped at
// its first failure: the sitemap came out cut short at round page counts.
export const createRequester = (concurrency: number): Requester => {
  let active = 0
  const waiters: Array<() => void> = []
  const acquire = async () => {
    if (active >= concurrency) {
      await new Promise<void>((resolve) => waiters.push(resolve))
    }
    active++
  }
  const release = () => {
    active--
    waiters.shift()?.()
  }
  return async (run) => {
    for (let attempt = 0; ; attempt++) {
      await acquire()
      let result
      try {
        result = await run()
      } finally {
        release()
      }
      const retryable =
        !result.ok &&
        (isTransientProblem(result.problem) || result.problem.status === 429)
      const delay = RETRY_DELAYS_MS[attempt]
      if (!retryable || delay === undefined) {
        return result
      }
      await new Promise((resolve) => setTimeout(resolve, delay))
    }
  }
}

const TOPIC_PAGE_LIMIT = 100
const TOPIC_MAX_PAGES = 200

export const collectTopicUrls = async (
  api: ApiClient,
  request: Requester = direct
): Promise<SitemapUrl[]> => {
  // GET /api/topic started returning { topics, total } on 2026-06-24, so
  // pick: Array.isArray was [] and the sitemap carried no topic URLs.
  const urls: SitemapUrl[] = []
  let cursor: string | undefined
  for (let page = 0; page < TOPIC_MAX_PAGES; page++) {
    const result = await request(() =>
      settle(
        api.GET('/topics', {
          params: {
            query: {
              limit: TOPIC_PAGE_LIMIT,
              ...(cursor ? { cursor } : {})
            }
          }
        })
      )
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
  api: ApiClient,
  request: Requester = direct
): Promise<SitemapUrl[]> => {
  const urls: SitemapUrl[] = []
  for (let page = 1; page <= ENTITY_MAX_PAGES; page++) {
    const result = await request(() =>
      settle(
        api.GET('/ratings', {
          params: { query: { page, limit: ENTITY_PAGE_LIMIT } }
        })
      )
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

type PagedBody<Item> = { items: Item[]; total: number }
type CursorBody<Item> = { items: Item[]; next_cursor?: string }

// Page-number collections refuse anything past 10,000 rows, so a source that
// can grow past that is read in its most popular order and stops at the cap.
const PAGED_LIMIT = 100
const PAGED_MAX_PAGES = 100

const collectPaged = async <Item>(
  fetchPage: (page: number) => Promise<PagedBody<Item> | null>,
  toUrl: (item: Item) => SitemapUrl,
  maxPages = PAGED_MAX_PAGES
): Promise<SitemapUrl[]> => {
  const urls: SitemapUrl[] = []
  for (let page = 1; page <= maxPages; page++) {
    const data = await fetchPage(page)
    if (!data) {
      break
    }
    urls.push(...data.items.map(toUrl))
    if (page * PAGED_LIMIT >= data.total || !data.items.length) {
      break
    }
  }
  return urls
}

const CURSOR_MAX_PAGES = 200

const collectCursor = async <Item>(
  fetchPage: (cursor?: string) => Promise<CursorBody<Item> | null>,
  toUrl: (item: Item) => SitemapUrl
): Promise<SitemapUrl[]> => {
  const urls: SitemapUrl[] = []
  let cursor: string | undefined
  for (let page = 0; page < CURSOR_MAX_PAGES; page++) {
    const data = await fetchPage(cursor)
    if (!data) {
      break
    }
    urls.push(...data.items.map(toUrl))
    if (!data.next_cursor) {
      break
    }
    cursor = data.next_cursor
  }
  return urls
}

const okData = <T>(result: { ok: true; data: T } | { ok: false }) =>
  result.ok ? result.data : null

const entry = (
  loc: string,
  priority: number,
  lastmod?: string | null
): SitemapUrl => ({
  loc,
  changefreq: 'daily',
  priority,
  ...(lastmod ? { lastmod } : {})
})

const WORK_PAGE_LIMIT = 50
const WORK_MAX_PAGES = 200
const range = (from: number, to: number): number[] =>
  Array.from({ length: Math.max(0, to - from + 1) }, (_, i) => from + i)

// The works the forum has a page for, and the makers behind them: catalog
// lists far more companies than a page-number read can reach, and the ones
// with a work here are the ones worth a crawl.
export const collectWorkUrls = async (
  api: ApiClient,
  request: Requester = direct
): Promise<SitemapUrl[]> => {
  const fetchPage = async (page: number) =>
    okData(
      await request(() =>
        settle(
          api.GET('/works', {
            params: {
              query: {
                page,
                limit: WORK_PAGE_LIMIT,
                sort: 'resource_updated_desc',
                include_resourceless: true,
                include_all_original_languages: true
              }
            }
          })
        )
      )
    )

  const first = await fetchPage(1)
  if (!first) {
    return []
  }
  const pages = Math.min(
    Math.ceil(first.total / WORK_PAGE_LIMIT),
    WORK_MAX_PAGES
  )
  const rest = await Promise.all(range(2, pages).map((page) => fetchPage(page)))
  const works = [first, ...rest].flatMap((body) => body?.items ?? [])

  const urls = works.map((work) =>
    entry(`/galgame/${work.id}`, 0.8, work.resource_updated_at)
  )
  for (const work of works) {
    if (work.maker) {
      urls.push(entry(`/galgame/official/${work.maker.id}`, 0.5))
    }
  }
  return urls
}

const CHARACTER_MAX_PAGES = 50

// Anonymous reads, so adult content stays out as it does for any SFW reader.
export const collectEntityUrls = async (
  api: ApiClient,
  request: Requester = direct
): Promise<SitemapUrl[]> => {
  const groups = await Promise.all([
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/galgame-resources', {
                params: {
                  query: { page, limit: PAGED_LIMIT, sort: 'download_desc' }
                }
              })
            )
          )
        ),
      (r) => entry(`/galgame/resource/${r.id}`, 0.6, r.edited_at)
    ),
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/tags', {
                params: { query: { page, limit: PAGED_LIMIT } }
              })
            )
          )
        ),
      (t) => entry(`/galgame/tag/${t.id}`, 0.5)
    ),
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/engines', {
                params: { query: { page, limit: PAGED_LIMIT } }
              })
            )
          )
        ),
      (e) => entry(`/galgame/engine/${e.id}`, 0.5)
    ),
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/series', {
                params: { query: { page, limit: PAGED_LIMIT } }
              })
            )
          )
        ),
      (s) => entry(`/galgame/series/${s.id}`, 0.5)
    ),
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/characters', {
                params: {
                  query: { page, limit: PAGED_LIMIT, sort: 'popularity_desc' }
                }
              })
            )
          )
        ),
      (c) => entry(`/galgame/character/${c.id}`, 0.4),
      CHARACTER_MAX_PAGES
    ),
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/toolsets', {
                params: { query: { page, limit: PAGED_LIMIT } }
              })
            )
          )
        ),
      (t) => entry(`/toolset/${t.id}`, 0.6, t.updated_at)
    ),
    collectPaged(
      async (page) =>
        okData(
          await request(() =>
            settle(
              api.GET('/quizzes', {
                params: { query: { page, limit: PAGED_LIMIT } }
              })
            )
          )
        ),
      (q) => entry(`/galgame-quiz/${q.id}`, 0.5, q.updated_at)
    ),
    collectCursor(
      async (cursor) =>
        okData(
          await request(() =>
            settle(
              api.GET('/news-items', {
                params: { query: { limit: 50, ...(cursor ? { cursor } : {}) } }
              })
            )
          )
        ),
      (n) => entry(`/news/${n.id}`, 0.6, n.published_at)
    ),
    collectCursor(
      async (cursor) =>
        okData(
          await request(() =>
            settle(
              api.GET('/docs', {
                params: { query: { limit: 100, ...(cursor ? { cursor } : {}) } }
              })
            )
          )
        ),
      (d) => entry(`/doc/${d.slug}`, 0.5, d.edited_at ?? d.published_at)
    ),
    collectCursor(
      async (cursor) =>
        okData(
          await request(() =>
            settle(
              api.GET('/websites', {
                params: { query: { limit: 100, ...(cursor ? { cursor } : {}) } }
              })
            )
          )
        ),
      (w) => entry(`/website/${w.host}`, 0.4)
    )
  ])
  return groups.flat()
}

export const buildSitemapUrls = async (
  apiBase: string
): Promise<SitemapUrl[]> => {
  const api = createApiClient({ origin: apiBase, timeoutMs: 15000 })
  const request = createRequester(6)
  const groups = await Promise.all([
    collectTopicUrls(api, request),
    collectWorkUrls(api, request),
    collectEntityUrls(api, request),
    collectRatingUrls(api, request)
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
