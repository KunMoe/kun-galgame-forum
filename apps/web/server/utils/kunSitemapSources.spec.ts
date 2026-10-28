import { describe, expect, it, vi } from 'vitest'
import { createApiClient } from '#shared/utils/api/client'
import {
  collectEntityUrls,
  collectTopicUrls,
  collectWorkUrls,
  createRequester
} from './kunSitemapSources'

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const topicItem = (id: string, bumpedAt: string) => ({
  object: 'topic',
  id,
  title: id,
  state: 'published',
  category: 'galgame',
  sections: [],
  cover_images: [],
  author: { object: 'user', id: '1', name: 'n', avatar: null },
  view_count: 0,
  like_count: 0,
  reply_count: 0,
  comment_count: 0,
  has_best_answer: false,
  mini_apps: [],
  is_nsfw: false,
  bumped_at: bumpedAt,
  created_at: '2020-01-01T00:00:00Z',
  upvoted_at: null
})

const listBody = (ids: Array<[string, string]>, next?: string) => ({
  object: 'list',
  items: ids.map(([id, bumped]) => topicItem(id, bumped)),
  ...(next ? { next_cursor: next } : {})
})

describe('collectTopicUrls', () => {
  it('walks three pages and emits every id once, in order, with lastmod equal to bumped_at', async () => {
    const pages = [
      listBody(
        [
          ['1', '2026-01-01T00:00:00.000Z'],
          ['2', '2026-01-02T00:00:00.000Z']
        ],
        'c1'
      ),
      listBody(
        [
          ['3', '2026-01-03T00:00:00.000Z'],
          ['4', '2026-01-04T00:00:00.000Z']
        ],
        'c2'
      ),
      listBody([['5', '2026-01-05T00:00:00.000Z']])
    ]
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(url.searchParams.get('limit')).toBe('100')
      expect(url.searchParams.get('include_nsfw')).toBeNull()
      expect(input.headers.get('cookie')).toBeNull()
      const cursor = url.searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(200, pages[0])
      }
      if (cursor === 'c1') {
        return jsonResponse(200, pages[1])
      }
      expect(cursor).toBe('c2')
      return jsonResponse(200, pages[2])
    })
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const urls = await collectTopicUrls(api)
    expect(urls.map((url) => url.loc)).toEqual([
      '/topic/1',
      '/topic/2',
      '/topic/3',
      '/topic/4',
      '/topic/5'
    ])
    expect(urls.map((url) => url.lastmod)).toEqual([
      '2026-01-01T00:00:00.000Z',
      '2026-01-02T00:00:00.000Z',
      '2026-01-03T00:00:00.000Z',
      '2026-01-04T00:00:00.000Z',
      '2026-01-05T00:00:00.000Z'
    ])
    expect(urls.every((url) => url.changefreq === 'daily')).toBe(true)
    expect(urls.every((url) => url.priority === 0.8)).toBe(true)
    expect(fetchSpy).toHaveBeenCalledTimes(3)
  })

  it('keeps page one URLs when page two fails', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const cursor = new URL(input.url).searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(
          200,
          listBody([['1', '2026-01-01T00:00:00.000Z']], 'c1')
        )
      }
      return jsonResponse(500, { error: 'nope' })
    })
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const urls = await collectTopicUrls(api)
    expect(urls).toEqual([
      {
        loc: '/topic/1',
        lastmod: '2026-01-01T00:00:00.000Z',
        changefreq: 'daily',
        priority: 0.8
      }
    ])
    expect(fetchSpy).toHaveBeenCalledTimes(2)
  })

  it('stops a cursor that never ends after 200 pages', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(
        200,
        listBody([['1', '2026-01-01T00:00:00.000Z']], 'forever')
      )
    )
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const urls = await collectTopicUrls(api)
    expect(fetchSpy).toHaveBeenCalledTimes(200)
    expect(urls).toHaveLength(200)
  })
})

const paged = (items: unknown[], total: number) => ({
  object: 'list',
  items,
  total,
  total_relation: 'eq'
})

describe('collectWorkUrls', () => {
  it('lists every work page and the maker behind each work', async () => {
    const work = (id: string, maker: string | null) => ({
      id,
      resource_updated_at: null,
      maker: maker ? { object: 'company', id: maker } : null
    })
    const fetchSpy = vi.fn(async (input: Request) => {
      const page = new URL(input.url).searchParams.get('page')
      return jsonResponse(
        200,
        page === '1'
          ? paged([work('1', '10'), work('2', null)], 60)
          : paged([work('3', '10')], 60)
      )
    })
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const locs = (await collectWorkUrls(api)).map((url) => url.loc)
    expect(locs).toEqual(
      expect.arrayContaining([
        '/galgame/1',
        '/galgame/2',
        '/galgame/3',
        '/galgame/official/10'
      ])
    )
    expect(fetchSpy).toHaveBeenCalledTimes(2)
  })
})

describe('collectEntityUrls', () => {
  it('reads each source in its popular order and builds its page URL', async () => {
    const seen: Record<string, URLSearchParams> = {}
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      const path = url.pathname.slice(url.pathname.lastIndexOf('/'))
      seen[path] = url.searchParams
      switch (path) {
        case '/galgame-resources':
          return jsonResponse(200, paged([{ id: '7', edited_at: null }], 1))
        case '/characters':
          return jsonResponse(200, paged([{ id: '8' }], 1))
        case '/news-items': {
          const cursor = url.searchParams.get('cursor')
          return jsonResponse(200, {
            object: 'list',
            items: [{ id: cursor ? '2' : '1', published_at: null }],
            ...(cursor ? {} : { next_cursor: 'n1' })
          })
        }
        case '/docs':
          return jsonResponse(200, {
            object: 'list',
            items: [{ slug: 'rules', edited_at: null, published_at: null }]
          })
        case '/websites':
          return jsonResponse(200, {
            object: 'list',
            items: [{ host: 'a.example' }]
          })
        default:
          return jsonResponse(200, paged([{ id: '9' }], 1))
      }
    })
    const api = createApiClient({
      origin: 'http://sitemap.test',
      fetch: fetchSpy
    })
    const locs = (await collectEntityUrls(api)).map((url) => url.loc)
    expect(locs).toEqual(
      expect.arrayContaining([
        '/galgame/resource/7',
        '/galgame/character/8',
        '/galgame/tag/9',
        '/galgame/engine/9',
        '/galgame/series/9',
        '/toolset/9',
        '/galgame-quiz/9',
        '/news/1',
        '/news/2',
        '/doc/rules',
        '/website/a.example'
      ])
    )
    expect(seen['/galgame-resources']?.get('sort')).toBe('download_desc')
    expect(seen['/characters']?.get('sort')).toBe('popularity_desc')
  })
})

describe('createRequester', () => {
  it('retries a rate-limited or transient failure, and not a missing one', async () => {
    vi.useFakeTimers()
    const request = createRequester(2)
    const limited = {
      ok: false as const,
      problem: {
        kind: 'problem' as const,
        status: 429,
        code: null,
        errors: [],
        requestId: null
      }
    }
    const run = vi
      .fn()
      .mockResolvedValueOnce(limited)
      .mockResolvedValueOnce({ ok: true as const, data: 1 })
    const pending = request(run)
    await vi.runAllTimersAsync()
    await expect(pending).resolves.toEqual({ ok: true, data: 1 })
    expect(run).toHaveBeenCalledTimes(2)

    const missing = vi.fn().mockResolvedValue({
      ok: false as const,
      problem: { ...limited.problem, status: 404 }
    })
    await request(missing)
    expect(missing).toHaveBeenCalledTimes(1)
    vi.useRealTimers()
  })
})
