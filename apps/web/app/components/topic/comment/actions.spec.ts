// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import type { Comment, CommentViewer, UserRef } from '#shared/utils/api/schemas'
import TopicCommentDelete from './Delete.vue'
import TopicCommentLike from './Like.vue'

const { useComponentMessageStore } = vi.hoisted(() => ({
  useComponentMessageStore: vi.fn(() => ({ alert: async () => true }))
}))

mockNuxtImport('useComponentMessageStore', () => useComponentMessageStore)

// A signed-in store starts the cloud-preferences sync, and through the typed
// client its GET /me/preferences lands in this spec's fetch stub first.
mockNuxtImport('useCloudPreferences', () => () => ({
  sync: async () => {},
  flush: async () => {}
}))

const user = (id: string): UserRef => ({
  object: 'user',
  id,
  name: `u${id}`,
  avatar: null
})

const commentViewer = (over: Partial<CommentViewer> = {}): CommentViewer => ({
  has_liked: false,
  can_edit: false,
  can_delete: false,
  can_like: false,
  ...over
})

const comment = (over: Partial<Comment> = {}): Comment => ({
  object: 'comment',
  id: '5',
  reply_id: '11',
  reply_floor: 2,
  parent_comment_id: null,
  author: user('2'),
  in_reply_to_user: user('3'),
  content: {
    object: 'document',
    children: [
      { object: 'paragraph', children: [{ object: 'text', value: 'hi' }] }
    ]
  },
  like_count: 0,
  created_at: '2026-01-01T00:00:00.000Z',
  edited_at: null,
  viewer: commentViewer(),
  ...over
})

type Captured = { method: string; url: string }

const captureFetch = (body: unknown, status = 200) => {
  const captured: Captured[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request | string | URL) => {
      const req = input instanceof Request ? input : new Request(String(input))
      captured.push({ method: req.method, url: req.url })
      return new Response(JSON.stringify(body), {
        status,
        headers: {
          'content-type':
            status >= 400 ? 'application/problem+json' : 'application/json'
        }
      })
    })
  )
  return captured
}

afterEach(() => {
  vi.unstubAllGlobals()
  usePersistUserStore().resetUser()
})

describe('the comment like slot is a slot, not a toggle', () => {
  it('likes with PUT and unlikes with DELETE', async () => {
    usePersistUserStore().id = 1
    const liking = captureFetch(
      comment({ like_count: 1, viewer: commentViewer({ has_liked: true }) })
    )
    const wrapper = await mountSuspended(TopicCommentLike, {
      props: { comment: comment({ viewer: commentViewer({ can_like: true }) }) }
    })
    await wrapper
      .findComponent({ name: 'KunReaction' })
      .vm.$emit('change', true)
    await vi.waitFor(() => expect(liking).toHaveLength(1))
    expect(liking[0]!.method).toBe('PUT')
    expect(liking[0]!.url).toContain('/comments/5/like')
    wrapper.unmount()

    const unliking = captureFetch(comment())
    const liked = await mountSuspended(TopicCommentLike, {
      props: {
        comment: comment({
          like_count: 1,
          viewer: commentViewer({ can_like: true, has_liked: true })
        })
      }
    })
    await liked.findComponent({ name: 'KunReaction' }).vm.$emit('change', false)
    await vi.waitFor(() => expect(unliking).toHaveLength(1))
    expect(unliking[0]!.method).toBe('DELETE')
    expect(unliking[0]!.url).toContain('/comments/5/like')
    liked.unmount()
  })

  it('sends nothing when viewer.can_like is false', async () => {
    usePersistUserStore().id = 2
    const captured = captureFetch(comment())
    const wrapper = await mountSuspended(TopicCommentLike, {
      props: {
        comment: comment({ viewer: commentViewer({ can_like: false }) })
      }
    })
    await wrapper
      .findComponent({ name: 'KunReaction' })
      .vm.$emit('change', true)
    expect(captured).toHaveLength(0)
    wrapper.unmount()
  })
})

describe('comment action buttons follow viewer.can_*', () => {
  it('hides delete when can_delete is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicCommentDelete, {
      props: {
        comment: comment({ viewer: commentViewer({ can_delete: false }) })
      }
    })
    expect(hidden.text()).not.toContain('删除评论')
    hidden.unmount()
    const shown = await mountSuspended(TopicCommentDelete, {
      props: {
        comment: comment({ viewer: commentViewer({ can_delete: true }) })
      }
    })
    expect(shown.text()).toContain('删除评论')
    shown.unmount()
  })

  it('deletes with DELETE /comments/{id}', async () => {
    usePersistUserStore().id = 2
    const captured = captureFetch(null, 204)
    const wrapper = await mountSuspended(TopicCommentDelete, {
      props: {
        comment: comment({ viewer: commentViewer({ can_delete: true }) })
      }
    })
    await wrapper.find('button').trigger('click')
    await vi.waitFor(() => expect(captured).toHaveLength(1))
    expect(captured[0]!.method).toBe('DELETE')
    expect(captured[0]!.url).toMatch(/\/comments\/5$/)
    wrapper.unmount()
  })
})
