// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import type { TopicEngagement } from '#shared/utils/api/schemas'
import { useReactions } from './useReactions'

const { reportProblem } = vi.hoisted(() => ({
  reportProblem: vi.fn()
}))

mockNuxtImport('reportProblem', () => reportProblem)

// A signed-in store starts the cloud-preferences sync, and through the typed
// client its GET /me/preferences lands in this spec's fetch stub first.
mockNuxtImport('useCloudPreferences', () => () => ({
  sync: async () => {},
  flush: async () => {}
}))

const engagement = (over: Partial<TopicEngagement> = {}): TopicEngagement => ({
  object: 'topic_engagement',
  topic_id: '42',
  like_count: 1,
  dislike_count: 0,
  favorite_count: 0,
  upvote_count: 0,
  upvoted_at: null,
  reactions: [
    {
      reaction: 'like',
      count: 1,
      reactors: [],
      viewer: { has_reacted: true }
    }
  ],
  viewer: {
    has_liked: true,
    has_disliked: false,
    has_favorited: false,
    has_upvoted: false,
    can_edit: false,
    can_hide: false,
    can_unhide: false,
    can_like: true,
    can_upvote: true,
    can_set_best_answer: false,
    can_pin_reply: false
  },
  ...over
})

const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' }
  })

const problemResponse = () =>
  new Response(
    JSON.stringify({
      type: 'about:blank',
      title: 'Error',
      status: 403,
      code: 'SELF_LIKE_FORBIDDEN',
      request_id: 'req_1',
      errors: []
    }),
    {
      status: 403,
      headers: { 'content-type': 'application/problem+json' }
    }
  )

const requestOf = (input: Request | string | URL) =>
  input instanceof Request ? input : new Request(String(input))

afterEach(() => {
  vi.unstubAllGlobals()
  reportProblem.mockReset()
  usePersistUserStore().resetUser()
})

describe('useReactions', () => {
  it('PUTs when the reaction is not mine and DELETEs when it is', async () => {
    const captured: { method: string; url: string }[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: Request | string | URL) => {
        const req = requestOf(input)
        captured.push({ method: req.method, url: req.url })
        return jsonResponse(200, engagement())
      })
    )
    usePersistUserStore().id = 2
    const scope = effectScope()
    const state = scope.run(() =>
      useReactions({
        topicId: 42,
        targetUserId: 9,
        reactions: [{ reaction: 'like', count: 1, mine: true, reactors: [] }]
      })
    )!
    await state.toggle('like')
    await state.toggle('heart')
    expect(captured[0]!.method).toBe('DELETE')
    expect(captured[0]!.url).toContain('/topics/42/reactions/like')
    expect(captured[1]!.method).toBe('PUT')
    expect(captured[1]!.url).toContain('/topics/42/reactions/heart')
    scope.stop()
  })

  it('switches like to dislike with one PUT and ends with the server list, and the callback receives the engagement', async () => {
    const captured: { method: string; url: string }[] = []
    const next = engagement({
      like_count: 0,
      dislike_count: 2,
      reactions: [
        {
          reaction: 'dislike',
          count: 2,
          reactors: [],
          viewer: { has_reacted: true }
        }
      ]
    })
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: Request | string | URL) => {
        const req = requestOf(input)
        captured.push({ method: req.method, url: req.url })
        return jsonResponse(200, next)
      })
    )
    usePersistUserStore().id = 2
    const onEngagement = vi.fn()
    const scope = effectScope()
    const state = scope.run(() =>
      useReactions({
        topicId: '42',
        targetUserId: 9,
        reactions: [{ reaction: 'like', count: 1, mine: true, reactors: [] }],
        onEngagement
      })
    )!
    await state.toggle('dislike')
    expect(captured).toHaveLength(1)
    expect(captured[0]!.method).toBe('PUT')
    expect(captured[0]!.url).toContain('/topics/42/reactions/dislike')
    expect(state.list.value).toEqual([
      { reaction: 'dislike', count: 2, mine: true, reactors: [] }
    ])
    expect(onEngagement).toHaveBeenCalledWith(next)
    scope.stop()
  })

  it('reverts the optimistic update when the write fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => problemResponse())
    )
    usePersistUserStore().id = 2
    const start = [{ reaction: 'heart', count: 3, mine: false, reactors: [] }]
    const scope = effectScope()
    const state = scope.run(() =>
      useReactions({
        replyId: '15',
        targetUserId: 9,
        reactions: start
      })
    )!
    await state.toggle('heart')
    expect(state.list.value).toEqual(start)
    expect(reportProblem).toHaveBeenCalledTimes(1)
    scope.stop()
  })
})
