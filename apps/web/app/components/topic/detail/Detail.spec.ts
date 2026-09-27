// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { Reply, Topic, UserRef } from '#shared/utils/api/schemas'
import TopicDetail from './Detail.vue'

const user = (): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null
})

const emptyDoc = (): Reply['content'] => ({ object: 'document', children: [] })

const reply = (
  id: string,
  floor: number,
  over: Partial<Reply> = {}
): Reply => ({
  object: 'reply',
  id,
  topic_id: '42',
  floor,
  author: user(),
  author_moemoepoint: 3,
  content: {
    object: 'document',
    children: [
      {
        object: 'paragraph',
        children: [{ object: 'text', value: `floor ${floor}` }]
      }
    ]
  },
  like_count: 0,
  dislike_count: 0,
  reactions: [],
  is_pinned: false,
  is_best_answer: false,
  comments: [],
  created_at: '2026-01-01T00:00:00.000Z',
  edited_at: null,
  viewer: null,
  ...over
})

const topic = (id: string, over: Partial<Topic> = {}): Topic => ({
  object: 'topic',
  id,
  title: `Topic ${id}`,
  access_scope: 'public',
  author: user(),
  author_moemoepoint: 10,
  best_answer: null,
  bumped_at: '2026-02-01T00:00:00.000Z',
  category: 'galgame',
  comment_count: 0,
  content: emptyDoc(),
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
  reply_count: 4,
  sections: ['g-chatting'],
  state: 'published',
  upvote_count: 0,
  upvoted_at: null,
  view_count: 1,
  viewer: null,
  ...over
})

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const requestOf = (input: Request | string | URL) =>
  input instanceof Request ? input : new Request(String(input))

const requestUrl = (input: Request | string | URL) =>
  new URL(requestOf(input).url)

const pathOf = (input: Request | string | URL) => requestUrl(input).pathname

const isRepliesList = (input: Request | string | URL) => {
  const path = pathOf(input)
  return path.includes('/topics/') && path.endsWith('/replies')
}

const isViews = (input: Request | string | URL) => {
  const req = requestOf(input)
  return pathOf(req).endsWith('/views') && req.method === 'POST'
}

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
  clearNuxtData()
})

const stubFetch = (replies: Reply[], commentFloor?: number) => {
  const fetchSpy = vi.fn(async (input: Request | string | URL) => {
    if (isViews(input)) {
      return new Response(null, { status: 204 })
    }
    if (isRepliesList(input)) {
      return jsonResponse(200, { object: 'list', items: replies })
    }
    if (commentFloor !== undefined && pathOf(input).includes('/comments/')) {
      return jsonResponse(200, { object: 'comment', reply_floor: commentFloor })
    }
    return jsonResponse(200, { code: 0, data: [] })
  })
  vi.stubGlobal('fetch', fetchSpy)
  return fetchSpy
}

const requestedFromFloors = (fetchSpy: ReturnType<typeof stubFetch>) =>
  fetchSpy.mock.calls
    .map((call) => call[0] as Request | string | URL)
    .filter((input) => isRepliesList(input))
    .map((input) => requestUrl(input).searchParams.get('from_floor'))

const replyIds = (root: VueWrapper) =>
  root.findAll('.kun-reply').map((node) => node.attributes('id') ?? '')

describe('TopicDetail', () => {
  it('renders the pinned reply and best answer once, above the list, and excludes them from the list', async () => {
    const pinned = reply('10', 10, { is_pinned: true })
    const best = reply('20', 20, { is_best_answer: true })
    const fetchSpy = stubFetch([pinned, reply('11', 11), best, reply('21', 21)])
    wrapper = await mountSuspended(TopicDetail, {
      props: {
        topic: topic('501', { pinned_reply: pinned, best_answer: best })
      },
      route: '/topic/501'
    })
    await vi.waitFor(() => {
      expect(replyIds(wrapper!).length).toBeGreaterThan(0)
    })
    const ids = replyIds(wrapper!)
    expect(ids.filter((id) => id.startsWith('10.')).length).toBe(1)
    expect(ids.filter((id) => id.startsWith('20.')).length).toBe(1)
    expect(ids.filter((id) => id.startsWith('11.')).length).toBe(1)
    expect(ids.filter((id) => id.startsWith('21.')).length).toBe(1)
    expect(ids[0]!.startsWith('10.')).toBe(true)
    expect(ids[1]!.startsWith('20.')).toBe(true)
    expect(
      fetchSpy.mock.calls.some((call) => isRepliesList(call[0] as Request))
    ).toBe(true)
  })

  it('loads the whole page holding the floor for ?reply=N', async () => {
    const fetchSpy = stubFetch([reply('45', 45)])
    wrapper = await mountSuspended(TopicDetail, {
      props: { topic: topic('502') },
      route: '/topic/502?reply=45'
    })
    await vi.waitFor(() => {
      expect(requestedFromFloors(fetchSpy)).toContain('31')
    })
  })

  it('loads the whole page holding the comment for ?comment=N', async () => {
    const fetchSpy = stubFetch([reply('4', 4)], 4)
    wrapper = await mountSuspended(TopicDetail, {
      props: { topic: topic('505') },
      route: '/topic/505?comment=3322'
    })
    await vi.waitFor(() => {
      expect(requestedFromFloors(fetchSpy)).toContain('1')
    })
  })

  it('sends the view beacon once on mount', async () => {
    const fetchSpy = stubFetch([])
    wrapper = await mountSuspended(TopicDetail, {
      props: { topic: topic('503') },
      route: '/topic/503'
    })
    await vi.waitFor(() => {
      expect(
        fetchSpy.mock.calls.filter((call) => isViews(call[0] as Request)).length
      ).toBe(1)
    })
    expect(
      fetchSpy.mock.calls.filter((call) => isViews(call[0] as Request))
    ).toHaveLength(1)
  })

  it('applies a created reply and an updated v1 Reply, and removes a deleted one', async () => {
    stubFetch([reply('11', 11)])
    wrapper = await mountSuspended(TopicDetail, {
      props: { topic: topic('504') },
      route: '/topic/504'
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('floor 11')
    })
    const store = useTempReplyStore()
    store.setSuccessfulReply({
      type: 'updated',
      data: reply('11', 11, {
        content: {
          object: 'document',
          children: [
            {
              object: 'paragraph',
              children: [{ object: 'text', value: 'edited body' }]
            }
          ]
        }
      })
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('edited body')
    })
    store.setSuccessfulReply({
      type: 'created',
      data: reply('12', 12)
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('floor 12')
    })
    store.setSuccessfulReply({
      type: 'deleted',
      data: { id: '11' }
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).not.toContain('edited body')
    })
    expect(wrapper!.text()).toContain('floor 12')
  })
})
