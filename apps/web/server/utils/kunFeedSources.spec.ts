import { describe, expect, it, vi } from 'vitest'
import { createApiClient } from '#shared/utils/api/client'
import {
  loadGalgameFeed,
  loadNewsFeed,
  loadToolsetFeed,
  loadTopicFeed
} from './kunFeedSources'

const origin = 'http://rss.test'
const baseUrl = 'https://www.kungal.com'

const jsonResponse = (status: number, body: unknown, contentType?: string) =>
  new Response(JSON.stringify(body), {
    status,
    headers: {
      'content-type': contentType ?? 'application/json'
    }
  })

const problem = (status: number) =>
  jsonResponse(
    status,
    {
      type: 'https://developer.nextmoe.dev/problems/platform/error',
      title: 'error',
      status,
      code: 'ERROR',
      errors: []
    },
    'application/problem+json'
  )

const user = (over: Record<string, unknown> = {}) => ({
  object: 'user',
  id: '8',
  name: 'Neko',
  avatar: null,
  ...over
})

const topicSummary = (over: Record<string, unknown> = {}) => ({
  object: 'topic',
  id: '42',
  title: 'A topic title',
  state: 'published',
  category: 'galgame',
  sections: ['g-chatting'],
  cover_images: [],
  author: user(),
  view_count: 0,
  like_count: 0,
  reply_count: 0,
  comment_count: 0,
  has_best_answer: false,
  mini_apps: [],
  is_nsfw: false,
  bumped_at: '2026-02-01T00:00:00.000Z',
  created_at: '2026-01-01T00:00:00.000Z',
  upvoted_at: null,
  ...over
})

const topicDetail = (over: Record<string, unknown> = {}) => ({
  ...topicSummary(),
  access_scope: 'public',
  author_moemoepoint: 1,
  best_answer: null,
  content: {
    object: 'document',
    children: [
      {
        object: 'paragraph',
        children: [{ object: 'text', value: 'plain excerpt' }]
      }
    ]
  },
  dislike_count: 0,
  edited_at: null,
  favorite_count: 0,
  hidden_by: null,
  pinned_reply: null,
  reactions: [],
  upvote_count: 0,
  viewer: null,
  ...over
})

const work = (over: Record<string, unknown> = {}) => ({
  object: 'work',
  id: '7',
  display_name: 'Original',
  latin: null,
  localized: {
    'zh-Hans': { value: '中文名', is_machine: false }
  },
  is_nsfw: false,
  cover: {
    hash: 'aa',
    url: 'https://cdn.example/cover.webp',
    width: 100,
    height: 100,
    thumbhash: null,
    sexual: null
  },
  ...over
})

const resource = (over: Record<string, unknown> = {}) => ({
  object: 'galgame_resource',
  id: '99',
  author: user(),
  comment_count: 0,
  content: { object: 'document', children: [] },
  created_at: '2026-03-01T00:00:00.000Z',
  dlsite: null,
  download_count: 0,
  edited_at: null,
  like_count: 0,
  provider_names: [],
  resource_languages: ['zh-cn'],
  resource_platforms: ['win'],
  resource_runtimes: ['native-win'],
  resource_type: 'game',
  size: '1.5 GB',
  state: 'valid',
  title: ' discrete title ',
  updated_at: '2026-04-01T00:00:00.000Z',
  version_label: 'official_latest',
  view_count: 0,
  viewer: null,
  work: work(),
  ...over
})

const pageList = (items: unknown[]) => ({
  object: 'list',
  items,
  total: items.length,
  total_relation: 'eq'
})

const cursorList = (items: unknown[]) => ({
  object: 'list',
  items
})

const client = (fetchSpy: (input: Request) => Promise<Response>) =>
  createApiClient({ origin, fetch: fetchSpy })

const segs = (url: URL) => url.pathname.split('/')

const expectMappedListErrors = async (
  run: (fetchSpy: (input: Request) => Promise<Response>) => Promise<unknown>
) => {
  expect(await run(async () => problem(404))).toEqual({
    ok: false,
    status: 404
  })
  expect(await run(async () => problem(400))).toEqual({
    ok: false,
    status: 404
  })
  expect(await run(async () => problem(503))).toEqual({
    ok: false,
    status: 502
  })
  expect(
    await run(async () => {
      throw new TypeError('fetch failed')
    })
  ).toEqual({ ok: false, status: 502 })
}

describe('loadTopicFeed', () => {
  it('requests created_desc limit 20 without include_nsfw and maps item fields', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'topics' && segs(url).length === 4) {
        expect(url.searchParams.get('sort')).toBe('created_desc')
        expect(url.searchParams.get('limit')).toBe('20')
        expect(url.searchParams.get('include_nsfw')).toBeNull()
        expect(url.searchParams.get('section')).toBeNull()
        return jsonResponse(200, cursorList([topicSummary()]))
      }
      expect(segs(url)).toEqual(['', 'api', 'v1', 'topics', '42'])
      return jsonResponse(200, topicDetail())
    })
    const result = await loadTopicFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('鲲 Galgame 论坛 - 新话题')
    expect(result.channel.description).toBe('鲲 Galgame 论坛最新发布的话题')
    expect(result.channel.link).toBe(`${baseUrl}/topic`)
    expect(result.channel.items).toHaveLength(1)
    const item = result.channel.items[0]!
    expect(item.link).toBe(`${baseUrl}/topic/42`)
    expect(item.title).toBe('A topic title')
    expect(item.date).toEqual(new Date('2026-01-01T00:00:00.000Z'))
    expect(item.description).toBe('plain excerpt')
    expect(item.author).toEqual([
      { name: 'Neko', link: `${baseUrl}/user/8/info` }
    ])
    expect(item.image).toBeUndefined()
  })

  it('sends include_nsfw=true and section when requested', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'topics' && segs(url).length === 4) {
        expect(url.searchParams.get('include_nsfw')).toBe('true')
        expect(url.searchParams.get('section')).toBe('g-chatting')
        return jsonResponse(200, cursorList([]))
      }
      throw new Error(url.pathname)
    })
    const result = await loadTopicFeed(client(fetchSpy), baseUrl, {
      includeNsfw: true,
      section: 'g-chatting'
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('鲲 Galgame 论坛 - Galgame · 闲聊 分区新话题')
    expect(result.channel.link).toBe(`${baseUrl}/section/g-chatting`)
  })

  it('keeps the item with an empty description when the detail read fails', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'topics' && segs(url).length === 4) {
        return jsonResponse(200, cursorList([topicSummary()]))
      }
      return problem(503)
    })
    const result = await loadTopicFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items[0]?.description).toBe('')
    expect(result.channel.items[0]?.link).toBe(`${baseUrl}/topic/42`)
  })

  it('uses 已注销用户 when the author name is null', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'topics' && segs(url).length === 4) {
        return jsonResponse(
          200,
          cursorList([topicSummary({ author: user({ name: null }) })])
        )
      }
      return jsonResponse(200, topicDetail())
    })
    const result = await loadTopicFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items[0]?.author[0]?.name).toBe('已注销用户')
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) =>
      loadTopicFeed(client(fetchSpy), baseUrl, { includeNsfw: false })
    )
  })
})

describe('loadGalgameFeed', () => {
  it('requests created_desc limit 50 without include_nsfw and maps a full resource', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(segs(url)).toEqual(['', 'api', 'v1', 'galgame-resources'])
      expect(url.searchParams.get('sort')).toBe('created_desc')
      expect(url.searchParams.get('limit')).toBe('50')
      expect(url.searchParams.get('include_nsfw')).toBeNull()
      return jsonResponse(
        200,
        pageList([
          resource({
            content: {
              object: 'document',
              children: [
                {
                  object: 'paragraph',
                  children: [{ object: 'text', value: 'a note' }]
                }
              ]
            }
          })
        ])
      )
    })
    const result = await loadGalgameFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('鲲 Galgame 论坛 - Galgame 新资源')
    expect(result.channel.description).toBe(
      '鲲 Galgame 论坛最新发布的 Galgame 下载资源'
    )
    expect(result.channel.link).toBe(`${baseUrl}/galgame/resource`)
    const item = result.channel.items[0]!
    expect(item.link).toBe(`${baseUrl}/galgame/resource/99`)
    expect(item.title).toBe('【游戏本体】中文名 · discrete title')
    expect(item.date).toEqual(new Date('2026-03-01T00:00:00.000Z'))
    expect(item.description).toBe(
      '语言: 简体中文 | 平台: Windows 电脑版 | 运行环境: Windows 原生 | 版本: 官方最新 | 大小: 1.5 GB\n\na note'
    )
    expect(item.author).toEqual([
      { name: 'Neko', link: `${baseUrl}/user/8/info` }
    ])
    expect(item.category).toEqual([{ name: '游戏本体' }])
    expect(item.image).toBe('https://cdn.example/cover.webp')
  })

  it('omits empty title, note, size, version, runtimes and platforms', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(
        200,
        pageList([
          resource({
            title: '   ',
            size: '  ',
            version_label: null,
            resource_runtimes: [],
            resource_platforms: [],
            resource_languages: ['ja-jp'],
            content: { object: 'document', children: [] }
          })
        ])
      )
    )
    const result = await loadGalgameFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    const item = result.channel.items[0]!
    expect(item.title).toBe('【游戏本体】中文名')
    expect(item.description).toBe('语言: 日语')
  })

  it('sends include_nsfw=true when requested', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      expect(new URL(input.url).searchParams.get('include_nsfw')).toBe('true')
      return jsonResponse(200, pageList([]))
    })
    await loadGalgameFeed(client(fetchSpy), baseUrl, { includeNsfw: true })
  })

  it('skips a resource whose work is null', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(
        200,
        pageList([resource({ work: null }), resource({ id: '100' })])
      )
    )
    const result = await loadGalgameFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items.map((item) => item.link)).toEqual([
      `${baseUrl}/galgame/resource/100`
    ])
  })

  it('withholds an NSFW work cover without include_nsfw and includes it when requested', async () => {
    const nsfw = resource({
      work: work({ is_nsfw: true })
    })
    const run = async (includeNsfw: boolean) => {
      const fetchSpy = vi.fn(async () => jsonResponse(200, pageList([nsfw])))
      return loadGalgameFeed(client(fetchSpy), baseUrl, { includeNsfw })
    }
    const hidden = await run(false)
    const shown = await run(true)
    expect(hidden.ok && hidden.channel.items[0]?.image).toBeUndefined()
    expect(shown.ok && shown.channel.items[0]?.image).toBe(
      'https://cdn.example/cover.webp'
    )
  })

  it('uses 已注销用户 when the author name is null', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(200, pageList([resource({ author: user({ name: null }) })]))
    )
    const result = await loadGalgameFeed(client(fetchSpy), baseUrl, {
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items[0]?.author[0]?.name).toBe('已注销用户')
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) =>
      loadGalgameFeed(client(fetchSpy), baseUrl, { includeNsfw: false })
    )
  })
})

describe('loadNewsFeed', () => {
  const source = {
    object: 'news_source',
    key: 'ymgal',
    display_name: '月幕',
    homepage_url: 'https://ymgal.games',
    attribution: '转载请注明月幕',
    column_url: '',
    forum_account: null
  }
  const item = {
    object: 'news_item',
    id: '1',
    lane: 'news',
    news_source: 'ymgal',
    preview: '  a preview  ',
    published_at: '2026-05-01T00:00:00.000Z',
    source_url: 'https://ymgal.games/1',
    title: '  Headline  '
  }

  it('requests limit 30 and /news-sources, maps fields, and appends attribution', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'news-items') {
        expect(url.searchParams.get('limit')).toBe('30')
        expect(url.searchParams.get('lane')).toBeNull()
        expect(url.searchParams.get('news_source')).toBeNull()
        return jsonResponse(
          200,
          cursorList([item, { ...item, news_source: 'missing', id: '2' }])
        )
      }
      expect(segs(url)).toEqual(['', 'api', 'v1', 'news-sources'])
      return jsonResponse(200, cursorList([source]))
    })
    const result = await loadNewsFeed(client(fetchSpy), baseUrl, {})
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('鲲 Galgame 论坛 - Gal 情报')
    expect(result.channel.description).toBe(
      '鲲 Galgame 论坛转载的 Galgame 情报与专栏, 版权归原作者所有'
    )
    expect(result.channel.link).toBe(`${baseUrl}/news`)
    expect(result.channel.items).toHaveLength(1)
    const mapped = result.channel.items[0]!
    expect(mapped.link).toBe('https://ymgal.games/1')
    expect(mapped.title).toBe('Headline')
    expect(mapped.date).toEqual(new Date('2026-05-01T00:00:00.000Z'))
    expect(mapped.description).toBe('a preview\n\n转载请注明月幕')
    expect(mapped.author).toEqual([
      { name: '月幕', link: 'https://ymgal.games' }
    ])
    expect(mapped.category).toEqual([{ name: '情报' }])
  })

  it('passes lane and news_source', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'news-items') {
        expect(url.searchParams.get('lane')).toBe('column')
        expect(url.searchParams.get('news_source')).toBe('ymgal')
        return jsonResponse(200, cursorList([]))
      }
      return jsonResponse(200, cursorList([source]))
    })
    await loadNewsFeed(client(fetchSpy), baseUrl, {
      lane: 'column',
      source: 'ymgal'
    })
  })

  it('fails with 502 when /news-sources returns 503 and 404 when it returns 404', async () => {
    const run = async (status: number) => {
      const fetchSpy = vi.fn(async (input: Request) => {
        const url = new URL(input.url)
        if (segs(url)[3] === 'news-items') {
          return jsonResponse(200, cursorList([item]))
        }
        return problem(status)
      })
      return loadNewsFeed(client(fetchSpy), baseUrl, {})
    }
    expect(await run(503)).toEqual({ ok: false, status: 502 })
    expect(await run(404)).toEqual({ ok: false, status: 404 })
  })

  it('omits attribution when the source leaves it empty', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'news-items') {
        return jsonResponse(200, cursorList([item]))
      }
      return jsonResponse(200, cursorList([{ ...source, attribution: '  ' }]))
    })
    const result = await loadNewsFeed(client(fetchSpy), baseUrl, {})
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items[0]?.description).toBe('a preview')
  })

  it('labels a column item as 专栏', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'news-items') {
        return jsonResponse(200, cursorList([{ ...item, lane: 'column' }]))
      }
      return jsonResponse(200, cursorList([source]))
    })
    const result = await loadNewsFeed(client(fetchSpy), baseUrl, {})
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items[0]?.category).toEqual([{ name: '专栏' }])
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) => {
      const wrapped = async (input: Request) => {
        const url = new URL(input.url)
        if (segs(url)[3] === 'news-sources') {
          return jsonResponse(200, cursorList([source]))
        }
        return fetchSpy(input)
      }
      return loadNewsFeed(client(wrapped), baseUrl, {})
    })
  })
})

describe('loadToolsetFeed', () => {
  it('requests created_desc limit 20 and maps item fields', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(segs(url)).toEqual(['', 'api', 'v1', 'toolsets'])
      expect(url.searchParams.get('sort')).toBe('created_desc')
      expect(url.searchParams.get('limit')).toBe('20')
      return jsonResponse(
        200,
        pageList([
          {
            object: 'toolset',
            id: '3',
            title: 'Krkr',
            aliases: ['kirikiri', 'krkr'],
            author: user(),
            comment_count: 0,
            created_at: '2026-02-02T00:00:00.000Z',
            download_count: 0,
            edited_at: null,
            homepage_urls: [],
            interface_language: 'zh-cn',
            platform: 'windows',
            practicality_average: null,
            practicality_count: 0,
            practicality_distribution: [0, 0, 0, 0, 0],
            release_channel: 'stable',
            resource_updated_at: '2026-07-01T00:00:00.000Z',
            toolset_type: 'extractor',
            updated_at: '2026-07-01T00:00:00.000Z',
            view_count: 0
          }
        ])
      )
    })
    const result = await loadToolsetFeed(client(fetchSpy), baseUrl)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('鲲 Galgame 论坛 - 新工具')
    expect(result.channel.description).toBe(
      '鲲 Galgame 论坛最新收录的 Galgame 工具'
    )
    expect(result.channel.link).toBe(`${baseUrl}/toolset`)
    const item = result.channel.items[0]!
    expect(item.link).toBe(`${baseUrl}/toolset/3`)
    expect(item.title).toBe('Krkr')
    expect(item.date).toEqual(new Date('2026-02-02T00:00:00.000Z'))
    expect(item.description).toBe(
      '类型: 解包工具 | 平台: Windows | 语言: 简体中文 | 别名: kirikiri / krkr'
    )
    expect(item.author).toEqual([
      { name: 'Neko', link: `${baseUrl}/user/8/info` }
    ])
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) =>
      loadToolsetFeed(client(fetchSpy), baseUrl)
    )
  })
})
