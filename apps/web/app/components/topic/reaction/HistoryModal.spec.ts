// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { Reaction, UserRef } from '#shared/utils/api/schemas'
import TopicReactionHistoryModal from './HistoryModal.vue'

const user = (): UserRef => ({
  object: 'user',
  id: '4',
  name: 'Dan',
  avatar: null,
  avatar_frame: null
})

const reaction = (id: string, over: Partial<Reaction> = {}): Reaction => ({
  object: 'reaction',
  id,
  reaction: 'like',
  reactor: user(),
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

const isTopicReactions = (input: Request | string | URL) => {
  const path = requestUrl(input).pathname
  return path.includes('/topics/') && path.endsWith('/reactions')
}

const isReplyReactions = (input: Request | string | URL) => {
  const path = requestUrl(input).pathname
  return path.includes('/replies/') && path.endsWith('/reactions')
}

let wrapper: VueWrapper | undefined

const bodyText = () => document.body.textContent ?? ''

const loadMoreButton = () =>
  [...document.body.querySelectorAll('button')].find((button) =>
    (button.textContent ?? '').includes('加载更多')
  )

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
  clearNuxtData()
})

describe('TopicReactionHistoryModal', () => {
  it('loads a topic first page, load more with the cursor, then the end', async () => {
    const fetchSpy = vi.fn(async (input: Request | string | URL) => {
      if (!isTopicReactions(input)) {
        return jsonResponse(200, {})
      }
      const cursor = requestUrl(input).searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(200, {
          object: 'list',
          items: [reaction('1')],
          next_cursor: 'c1'
        })
      }
      return jsonResponse(200, { object: 'list', items: [reaction('2')] })
    })
    vi.stubGlobal('fetch', fetchSpy)
    wrapper = await mountSuspended(TopicReactionHistoryModal, {
      props: { topicId: '42', modelValue: true }
    })
    await vi.waitFor(() => {
      expect(bodyText()).toContain('加载更多')
    })
    expect(bodyText()).toContain('Dan')
    loadMoreButton()!.click()
    await vi.waitFor(() => {
      expect(
        fetchSpy.mock.calls.some((call) => {
          const input = call[0] as Request | string | URL
          return (
            isTopicReactions(input) &&
            requestUrl(input).searchParams.get('cursor') === 'c1'
          )
        })
      ).toBe(true)
    })
    await vi.waitFor(() => {
      expect(bodyText()).not.toContain('加载更多')
    })
  })

  it('loads a reply first page, load more with the cursor, then the end', async () => {
    const fetchSpy = vi.fn(async (input: Request | string | URL) => {
      if (!isReplyReactions(input)) {
        return jsonResponse(200, {})
      }
      const cursor = requestUrl(input).searchParams.get('cursor')
      if (!cursor) {
        return jsonResponse(200, {
          object: 'list',
          items: [reaction('8')],
          next_cursor: 'r1'
        })
      }
      return jsonResponse(200, { object: 'list', items: [reaction('9')] })
    })
    vi.stubGlobal('fetch', fetchSpy)
    wrapper = await mountSuspended(TopicReactionHistoryModal, {
      props: { replyId: '15', modelValue: true }
    })
    await vi.waitFor(() => {
      expect(bodyText()).toContain('加载更多')
    })
    loadMoreButton()!.click()
    await vi.waitFor(() => {
      expect(
        fetchSpy.mock.calls.some((call) => {
          const input = call[0] as Request | string | URL
          return (
            isReplyReactions(input) &&
            requestUrl(input).searchParams.get('cursor') === 'r1'
          )
        })
      ).toBe(true)
    })
    await vi.waitFor(() => {
      expect(bodyText()).not.toContain('加载更多')
    })
  })
})
