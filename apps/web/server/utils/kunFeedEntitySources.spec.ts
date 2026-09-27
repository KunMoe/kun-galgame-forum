import { describe, expect, it, vi } from 'vitest'
import { createApiClient } from '#shared/utils/api/client'
import {
  loadUserGalgameFeed,
  loadUserTopicFeed,
  loadWorkResourceFeed
} from './kunFeedEntitySources'

const origin = 'http://rss.test'
const baseUrl = 'https://www.kungal.com'

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const problem = (status: number) =>
  jsonResponse(status, {
    type: 'https://developer.nextmoe.dev/problems/platform/error',
    title: 'error',
    status,
    code: 'ERROR',
    errors: []
  })

const user = (over: Record<string, unknown> = {}) => ({
  object: 'user',
  id: '8',
  name: 'Neko',
  avatar: null,
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
  resource_runtimes: [],
  resource_type: 'game',
  size: '1 GB',
  state: 'valid',
  title: '',
  updated_at: '2026-04-01T00:00:00.000Z',
  version_label: null,
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

describe('loadWorkResourceFeed', () => {
  it('requests state=valid limit 50 and does not send include_nsfw', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(segs(url)).toEqual(['', 'api', 'v1', 'works', '7', 'resources'])
      expect(url.searchParams.get('state')).toBe('valid')
      expect(url.searchParams.get('limit')).toBe('50')
      expect(url.searchParams.get('include_nsfw')).toBeNull()
      return jsonResponse(200, pageList([resource()]))
    })
    const result = await loadWorkResourceFeed(client(fetchSpy), baseUrl, {
      workId: '7',
      includeNsfw: true
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('中文名 的新资源 - 鲲 Galgame 论坛')
    expect(result.channel.description).toBe(
      '鲲 Galgame 论坛上这部作品新发布的下载资源'
    )
    expect(result.channel.link).toBe(`${baseUrl}/galgame/7`)
    expect(result.channel.image).toBe('https://cdn.example/cover.webp')
    expect(result.channel.items[0]?.link).toBe(`${baseUrl}/galgame/resource/99`)
    expect(result.channel.items[0]?.image).toBe(
      'https://cdn.example/cover.webp'
    )
  })

  it('uses the fallback channel title when the list is empty', async () => {
    const fetchSpy = vi.fn(async () => jsonResponse(200, pageList([])))
    const result = await loadWorkResourceFeed(client(fetchSpy), baseUrl, {
      workId: '7',
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('Galgame 7 的新资源 - 鲲 Galgame 论坛')
    expect(result.channel.image).toBe(`${baseUrl}/kungalgame.webp`)
    expect(result.channel.items).toEqual([])
  })

  it('withholds NSFW cover on the item and channel without include_nsfw', async () => {
    const nsfw = resource({ work: work({ is_nsfw: true }) })
    const hidden = await loadWorkResourceFeed(
      client(async () => jsonResponse(200, pageList([nsfw]))),
      baseUrl,
      { workId: '7', includeNsfw: false }
    )
    const shown = await loadWorkResourceFeed(
      client(async () => jsonResponse(200, pageList([nsfw]))),
      baseUrl,
      { workId: '7', includeNsfw: true }
    )
    expect(hidden.ok && hidden.channel.items[0]?.image).toBeUndefined()
    expect(hidden.ok && hidden.channel.image).toBe(`${baseUrl}/kungalgame.webp`)
    expect(shown.ok && shown.channel.items[0]?.image).toBe(
      'https://cdn.example/cover.webp'
    )
    expect(shown.ok && shown.channel.image).toBe(
      'https://cdn.example/cover.webp'
    )
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) =>
      loadWorkResourceFeed(client(fetchSpy), baseUrl, {
        workId: '7',
        includeNsfw: false
      })
    )
  })
})

describe('loadUserTopicFeed', () => {
  const userTopic = {
    object: 'topic',
    id: '42',
    title: 'A topic title',
    created_at: '2026-01-01T00:00:00.000Z'
  }
  const detail = {
    object: 'topic',
    id: '42',
    title: 'A topic title',
    access_scope: 'public',
    author: user(),
    author_moemoepoint: 1,
    best_answer: null,
    bumped_at: '2026-02-01T00:00:00.000Z',
    category: 'galgame',
    comment_count: 0,
    content: {
      object: 'document',
      children: [
        {
          object: 'paragraph',
          children: [{ object: 'text', value: 'plain excerpt' }]
        }
      ]
    },
    cover_images: [],
    created_at: '2026-01-01T00:00:00.000Z',
    dislike_count: 0,
    edited_at: null,
    favorite_count: 0,
    hidden_by: null,
    is_nsfw: false,
    like_count: 0,
    mini_apps: [],
    pinned_reply: null,
    reactions: [],
    reply_count: 0,
    sections: [],
    state: 'published',
    upvote_count: 0,
    upvoted_at: null,
    view_count: 0,
    viewer: null
  }

  it('requests relation=authored limit 20 and include_nsfw when asked', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'users') {
        expect(url.searchParams.get('relation')).toBe('authored')
        expect(url.searchParams.get('limit')).toBe('20')
        expect(url.searchParams.get('include_nsfw')).toBe('true')
        return jsonResponse(200, pageList([userTopic]))
      }
      expect(segs(url)).toEqual(['', 'api', 'v1', 'topics', '42'])
      return jsonResponse(200, detail)
    })
    const result = await loadUserTopicFeed(client(fetchSpy), baseUrl, {
      userId: '8',
      includeNsfw: true
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('Neko 的话题 - 鲲 Galgame 论坛')
    expect(result.channel.description).toBe('Neko 在鲲 Galgame 论坛发布的话题')
    expect(result.channel.link).toBe(`${baseUrl}/user/8/info`)
    expect(result.channel.items[0]?.link).toBe(`${baseUrl}/topic/42`)
    expect(result.channel.items[0]?.description).toBe('plain excerpt')
    expect(result.channel.items[0]?.author).toEqual([
      { name: 'Neko', link: `${baseUrl}/user/8/info` }
    ])
  })

  it('omits include_nsfw when not requested and uses the empty-list channel title', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(segs(url)).toEqual(['', 'api', 'v1', 'users', '8', 'topics'])
      expect(url.searchParams.get('include_nsfw')).toBeNull()
      return jsonResponse(200, pageList([]))
    })
    const result = await loadUserTopicFeed(client(fetchSpy), baseUrl, {
      userId: '8',
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe('用户 8 的话题 - 鲲 Galgame 论坛')
    expect(result.channel.items).toEqual([])
  })

  it('keeps the item with an empty description when the detail read fails', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      if (segs(url)[3] === 'users') {
        return jsonResponse(200, pageList([userTopic]))
      }
      return problem(503)
    })
    const result = await loadUserTopicFeed(client(fetchSpy), baseUrl, {
      userId: '8',
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.items[0]?.description).toBe('')
    expect(result.channel.items[0]?.link).toBe(`${baseUrl}/topic/42`)
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) =>
      loadUserTopicFeed(client(fetchSpy), baseUrl, {
        userId: '8',
        includeNsfw: false
      })
    )
  })
})

describe('loadUserGalgameFeed', () => {
  it('requests relation=published limit 50 and maps items', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      const url = new URL(input.url)
      expect(segs(url)).toEqual([
        '',
        'api',
        'v1',
        'users',
        '8',
        'galgame-resources'
      ])
      expect(url.searchParams.get('relation')).toBe('published')
      expect(url.searchParams.get('limit')).toBe('50')
      expect(url.searchParams.get('include_nsfw')).toBeNull()
      return jsonResponse(200, pageList([resource()]))
    })
    const result = await loadUserGalgameFeed(client(fetchSpy), baseUrl, {
      userId: '8',
      includeNsfw: false
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe(
      'Neko 发布的 Galgame 资源 - 鲲 Galgame 论坛'
    )
    expect(result.channel.description).toBe(
      'Neko 在鲲 Galgame 论坛发布的 Galgame 下载资源'
    )
    expect(result.channel.link).toBe(`${baseUrl}/user/8/resource/valid`)
    expect(result.channel.items[0]?.link).toBe(`${baseUrl}/galgame/resource/99`)
  })

  it('names a user with a null name as deleted, not by id', async () => {
    const fetchSpy = vi.fn(async () =>
      jsonResponse(200, pageList([resource({ author: user({ name: null }) })]))
    )
    const result = await loadUserGalgameFeed(client(fetchSpy), baseUrl, {
      userId: '8',
      includeNsfw: false
    })
    expect(result.ok && result.channel.title).toBe(
      '已注销用户 发布的 Galgame 资源 - 鲲 Galgame 论坛'
    )
  })

  it('sends include_nsfw=true and uses the empty-list channel title', async () => {
    const fetchSpy = vi.fn(async (input: Request) => {
      expect(new URL(input.url).searchParams.get('include_nsfw')).toBe('true')
      return jsonResponse(200, pageList([]))
    })
    const result = await loadUserGalgameFeed(client(fetchSpy), baseUrl, {
      userId: '8',
      includeNsfw: true
    })
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.channel.title).toBe(
      '用户 8 发布的 Galgame 资源 - 鲲 Galgame 论坛'
    )
  })

  it('maps list 404 and 400 to 404, 503 and network errors to 502', async () => {
    await expectMappedListErrors((fetchSpy) =>
      loadUserGalgameFeed(client(fetchSpy), baseUrl, {
        userId: '8',
        includeNsfw: false
      })
    )
  })
})
