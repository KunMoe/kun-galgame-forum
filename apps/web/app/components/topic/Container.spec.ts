// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { TopicSummary, UserRef } from '#shared/utils/api/schemas'
import TopicContainer from './Container.vue'

const user = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null,
  ...over
})

const topic = (id: string, over: Partial<TopicSummary> = {}): TopicSummary => ({
  object: 'topic',
  id,
  title: `Topic ${id}`,
  state: 'published',
  category: 'galgame',
  sections: ['g-chatting'],
  cover_images: [],
  author: user(),
  view_count: 10,
  like_count: 0,
  reply_count: 0,
  comment_count: 0,
  has_best_answer: false,
  mini_apps: [],
  is_nsfw: false,
  bumped_at: '2026-09-01T00:00:00.000Z',
  created_at: '2026-08-01T00:00:00.000Z',
  edited_at: null,
  upvoted_at: null,
  ...over
})

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const listBody = (items: TopicSummary[], next?: string) => ({
  object: 'list',
  items,
  ...(next ? { next_cursor: next } : {})
})

const requestUrl = (input: Request | string | URL) =>
  new URL(input instanceof Request ? input.url : String(input))

const isTopicsRequest = (input: Request | string | URL) =>
  requestUrl(input).pathname.split('/').filter(Boolean).at(-1) === 'topics'

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
  usePersistSettingsStore().showKUNGalgameContentLimit = 'sfw'
  clearNuxtData()
})

const stubTopics = (handler: (url: URL) => unknown) => {
  const fetchSpy = vi.fn(async (input: Request | string | URL) => {
    if (!isTopicsRequest(input)) {
      return jsonResponse(200, {})
    }
    return jsonResponse(200, handler(requestUrl(input)))
  })
  vi.stubGlobal('fetch', fetchSpy)
  return fetchSpy
}

const topicCalls = (fetchSpy: ReturnType<typeof stubTopics>) =>
  fetchSpy.mock.calls
    .map(([input]) => input as Request | string | URL)
    .filter((input) => isTopicsRequest(input))
    .map((input) => requestUrl(input))

describe('TopicContainer', () => {
  it('requests sort=bumped_desc when the URL sort is not offered', async () => {
    const fetchSpy = stubTopics(() => listBody([topic('1')], 'c1'))
    wrapper = await mountSuspended(TopicContainer, {
      route: '/topic?sort=garbage'
    })

    await vi.waitFor(() => {
      expect(topicCalls(fetchSpy).length).toBeGreaterThan(0)
    })
    const url = topicCalls(fetchSpy)[0]!
    expect(url.searchParams.get('sort')).toBe('bumped_desc')
    expect(url.searchParams.get('limit')).toBe('50')
  })

  it('requests include_nsfw=true when the store says nsfw', async () => {
    const fetchSpy = stubTopics(() => listBody([topic('1')]))
    usePersistSettingsStore().showKUNGalgameContentLimit = 'nsfw'
    wrapper = await mountSuspended(TopicContainer, { route: '/topic' })

    await vi.waitFor(() => {
      expect(topicCalls(fetchSpy).length).toBeGreaterThan(0)
    })
    expect(topicCalls(fetchSpy)[0]!.searchParams.get('include_nsfw')).toBe(
      'true'
    )
  })

  it('switches sort in the URL and the request when a sort control changes', async () => {
    const fetchSpy = stubTopics(() => listBody([topic('1')]))
    wrapper = await mountSuspended(TopicContainer, { route: '/topic' })
    await vi.waitFor(() => {
      expect(topicCalls(fetchSpy)).toHaveLength(1)
    })

    const buttons = wrapper
      .findAll('button')
      .filter(
        (button) =>
          !button.text().includes('更新时间') &&
          !button.text().includes('加载更多') &&
          !button.text().includes('没有更多')
      )
    await buttons[1]!.trigger('click')

    await vi.waitFor(() => {
      expect(useRoute().query.sort).toBe('bumped_asc')
    })
    await vi.waitFor(() => {
      expect(
        topicCalls(fetchSpy).some(
          (url) => url.searchParams.get('sort') === 'bumped_asc'
        )
      ).toBe(true)
    })
  })

  it('shows 加载更多 while next_cursor is present and 没有更多话题了 after the last page', async () => {
    const fetchSpy = stubTopics((url) => {
      const cursor = url.searchParams.get('cursor')
      if (!cursor) {
        return listBody([topic('1')], 'c1')
      }
      return listBody([topic('2')])
    })
    wrapper = await mountSuspended(TopicContainer, { route: '/topic' })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('加载更多')
    })
    expect(wrapper.text()).not.toContain('没有更多话题了')

    const more = wrapper
      .findAll('button')
      .find((button) => button.text().includes('加载更多'))
    expect(more).toBeDefined()
    await more!.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('没有更多话题了')
    })
    expect(wrapper.text()).not.toContain('加载更多')
    const urls = topicCalls(fetchSpy)
    expect(urls[0]!.searchParams.get('cursor')).toBeNull()
    expect(urls.some((url) => url.searchParams.get('cursor') === 'c1')).toBe(
      true
    )
    expect(
      urls
        .find((url) => url.searchParams.get('cursor') === 'c1')!
        .searchParams.get('limit')
    ).toBe('50')
  })
})
