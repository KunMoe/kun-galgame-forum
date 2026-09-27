// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { TopicUpvote, UserRef } from '#shared/utils/api/schemas'
import TopicUpvoteRecords from './UpvoteRecords.vue'

const user = (): UserRef => ({
  object: 'user',
  id: '3',
  name: 'Cara',
  avatar: null,
  avatar_frame: null
})

const upvote = (id: string, over: Partial<TopicUpvote> = {}): TopicUpvote => ({
  object: 'topic_upvote',
  id,
  topic_id: '42',
  upvoter: user(),
  note: null,
  created_at: '2026-01-01T00:00:00.000Z',
  ...over
})

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const requestUrl = (input: Request | string | URL) =>
  new URL(input instanceof Request ? input.url : String(input))

const isUpvotes = (input: Request | string | URL) =>
  requestUrl(input).pathname.endsWith('/upvotes') &&
  (input instanceof Request ? input.method === 'GET' : true)

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
  clearNuxtData()
})

describe('TopicUpvoteRecords', () => {
  it('loads the first page, load more with the cursor, then the end', async () => {
    const fetchSpy = vi.fn(async (input: Request | string | URL) => {
      if (!isUpvotes(input)) {
        return jsonResponse(200, {})
      }
      const cursor = requestUrl(input).searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(200, {
          object: 'list',
          items: [upvote('1')],
          next_cursor: 'c1'
        })
      }
      return jsonResponse(200, { object: 'list', items: [upvote('2')] })
    })
    vi.stubGlobal('fetch', fetchSpy)
    wrapper = await mountSuspended(TopicUpvoteRecords, {
      props: { topicId: '42' }
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('加载更多')
    })
    expect(wrapper!.text()).toContain('Cara')
    const more = wrapper!
      .findAll('button')
      .find((button) => button.text().includes('加载更多'))
    await more!.trigger('click')
    await vi.waitFor(() => {
      expect(
        fetchSpy.mock.calls.some((call) => {
          const input = call[0] as Request | string | URL
          return (
            isUpvotes(input) &&
            requestUrl(input).searchParams.get('cursor') === 'c1'
          )
        })
      ).toBe(true)
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).not.toContain('加载更多')
    })
  })
})
