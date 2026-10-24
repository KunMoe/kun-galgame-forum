// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { TopicSummary, UserRef } from '#shared/utils/api/schemas'
import TopicContainer from './Container.vue'

const { sentinelVisible } = vi.hoisted(() => ({
  sentinelVisible: { value: false }
}))
vi.mock('@vueuse/core', async (original) => {
  const { ref } = await import('vue')
  const visible = ref(false)
  Object.defineProperty(sentinelVisible, 'value', {
    get: () => visible.value,
    set: (v: boolean) => {
      visible.value = v
    }
  })
  return {
    ...(await original<typeof import('@vueuse/core')>()),
    useElementVisibility: () => visible
  }
})

mockNuxtImport('useTopicSections', () => () => {
  const discussion = {
    section: 'galgame-discussion',
    label: 'Galgame 讨论',
    color: 'primary',
    icon_name: 'lucide:messages-square'
  }
  const sectionOf = (key?: string | null) =>
    key === discussion.section ? discussion : undefined
  return {
    sectionOf,
    sectionLabel: (key?: string | null) => sectionOf(key)?.label ?? ''
  }
})

const user = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null,
  avatar_frame: null,
  ...over
})

const topic = (id: string, over: Partial<TopicSummary> = {}): TopicSummary => ({
  object: 'topic',
  id,
  title: `Topic ${id}`,
  state: 'published',
  category: 'galgame',
  section: 'galgame-discussion',
  ai_summary: null,
  sections: ['galgame-discussion'],
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

  it('loads the next page once the end of the list scrolls into view, then says 没有更多话题了', async () => {
    const fetchSpy = stubTopics((url) => {
      const cursor = url.searchParams.get('cursor')
      if (!cursor) {
        return listBody([topic('1')], 'c1')
      }
      return listBody([topic('2')])
    })
    sentinelVisible.value = false
    wrapper = await mountSuspended(TopicContainer, { route: '/topic' })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('正在加载更多话题')
    })
    expect(
      topicCalls(fetchSpy).some((url) => url.searchParams.get('cursor'))
    ).toBe(false)

    sentinelVisible.value = true
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('没有更多话题了')
    })
    sentinelVisible.value = false
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
