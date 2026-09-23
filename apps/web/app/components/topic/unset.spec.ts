// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import type {
  Reply,
  ReplyViewer,
  Topic,
  TopicViewer,
  UserRef
} from '#shared/utils/api/schemas'
import TopicFooterFavorite from './footer/Favorite.vue'
import TopicReplyBestAnswer from './reply/BestAnswer.vue'
import TopicReplyPin from './reply/Pin.vue'

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

const user = (): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null
})

const topicViewer = (over: Partial<TopicViewer> = {}): TopicViewer => ({
  has_liked: false,
  has_disliked: false,
  has_favorited: false,
  has_upvoted: false,
  can_edit: false,
  can_hide: false,
  can_unhide: false,
  can_like: false,
  can_upvote: false,
  can_set_best_answer: false,
  can_pin_reply: false,
  ...over
})

const topic = (over: Partial<Topic> = {}): Topic => ({
  object: 'topic',
  id: '42',
  title: 'Topic',
  access_scope: 'public',
  author: user(),
  author_moemoepoint: 10,
  best_answer: null,
  bumped_at: '2026-02-01T00:00:00.000Z',
  category: 'galgame',
  comment_count: 0,
  content: { object: 'document', children: [] },
  cover_images: [],
  created_at: '2026-01-01T00:00:00.000Z',
  dislike_count: 0,
  edited_at: null,
  favorite_count: 1,
  hidden_by: null,
  is_nsfw: false,
  like_count: 0,
  mini_apps: [],
  pinned_reply: null,
  reactions: [],
  reply_count: 1,
  sections: ['g-chatting'],
  state: 'published',
  upvote_count: 0,
  upvoted_at: null,
  view_count: 1,
  viewer: topicViewer(),
  ...over
})

const replyViewer = (over: Partial<ReplyViewer> = {}): ReplyViewer => ({
  can_edit: false,
  can_delete: false,
  can_like: false,
  has_liked: false,
  has_disliked: false,
  ...over
})

const reply = (over: Partial<Reply> = {}): Reply => ({
  object: 'reply',
  id: '11',
  topic_id: '42',
  floor: 2,
  author: user(),
  author_moemoepoint: 3,
  content: { object: 'document', children: [] },
  like_count: 0,
  dislike_count: 0,
  reactions: [],
  is_pinned: false,
  is_best_answer: false,
  comments: [],
  created_at: '2026-01-01T00:00:00.000Z',
  edited_at: null,
  viewer: replyViewer(),
  ...over
})

const captureFetch = (body: unknown) => {
  const captured: { method: string; url: string }[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request | string | URL) => {
      const req = input instanceof Request ? input : new Request(String(input))
      captured.push({ method: req.method, url: req.url })
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'content-type': 'application/json' }
      })
    })
  )
  return captured
}

afterEach(() => {
  vi.unstubAllGlobals()
  usePersistUserStore().resetUser()
})

describe('unsetting a viewer slot sends DELETE', () => {
  it('unfavorites with DELETE and favorites with PUT', async () => {
    const captured = captureFetch({
      object: 'topic_engagement',
      topic_id: '42',
      like_count: 0,
      dislike_count: 0,
      favorite_count: 0,
      upvote_count: 0,
      upvoted_at: null,
      reactions: [],
      viewer: topicViewer()
    })
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(TopicFooterFavorite, {
      props: { topic: topic({ viewer: topicViewer({ has_favorited: true }) }) }
    })
    await wrapper
      .findComponent({ name: 'KunReaction' })
      .vm.$emit('change', false)
    expect(captured[0]!.method).toBe('DELETE')
    expect(captured[0]!.url).toContain('/topics/42/favorite')
    wrapper.unmount()

    const second = captureFetch({
      object: 'topic_engagement',
      topic_id: '42',
      like_count: 0,
      dislike_count: 0,
      favorite_count: 1,
      upvote_count: 0,
      upvoted_at: null,
      reactions: [],
      viewer: topicViewer({ has_favorited: true })
    })
    const fresh = await mountSuspended(TopicFooterFavorite, {
      props: { topic: topic({ viewer: topicViewer({ has_favorited: false }) }) }
    })
    await fresh.findComponent({ name: 'KunReaction' }).vm.$emit('change', true)
    expect(second[0]!.method).toBe('PUT')
    fresh.unmount()
  })

  // The write answers with the whole engagement snapshot; applying only the
  // two fields the button changed left the other counts stale on the page.
  it('applies every field of the engagement snapshot, not just the count it changed', async () => {
    const snapshot = {
      object: 'topic_engagement',
      topic_id: '42',
      like_count: 7,
      dislike_count: 2,
      favorite_count: 5,
      upvote_count: 3,
      upvoted_at: '2026-03-01T00:00:00.000Z',
      reactions: [
        {
          reaction: 'like',
          count: 7,
          reactors: [],
          viewer: { has_reacted: true }
        }
      ],
      viewer: topicViewer({ has_favorited: true, has_liked: true })
    }
    captureFetch(snapshot)
    usePersistUserStore().id = 1
    const replaced: Topic[] = []
    const wrapper = await mountSuspended(TopicFooterFavorite, {
      props: {
        topic: topic({ viewer: topicViewer({ has_favorited: false }) })
      },
      global: { provide: { replaceTopic: (t: Topic) => replaced.push(t) } }
    })
    await wrapper
      .findComponent({ name: 'KunReaction' })
      .vm.$emit('change', true)
    await vi.waitFor(() => expect(replaced).toHaveLength(1))
    const next = replaced[0]!
    expect(next.like_count).toBe(7)
    expect(next.dislike_count).toBe(2)
    expect(next.favorite_count).toBe(5)
    expect(next.upvote_count).toBe(3)
    expect(next.upvoted_at).toBe('2026-03-01T00:00:00.000Z')
    expect(next.reactions).toHaveLength(1)
    expect(next.viewer?.has_favorited).toBe(true)
    expect(next.title).toBe('Topic')
    wrapper.unmount()
  })

  it('clears the best answer with DELETE and sets it with PUT', async () => {
    const provide = {
      pageTopic: computed(() =>
        topic({ viewer: topicViewer({ can_set_best_answer: true }) })
      ),
      replaceTopic: () => undefined
    }
    const captured = captureFetch(topic())
    const clearing = await mountSuspended(TopicReplyBestAnswer, {
      props: { reply: reply({ is_best_answer: true }) },
      global: { provide }
    })
    await clearing.find('button').trigger('click')
    expect(captured[0]!.method).toBe('DELETE')
    expect(captured[0]!.url).toContain('/topics/42/best-answer')
    clearing.unmount()

    const setting = captureFetch(topic())
    const wrapper = await mountSuspended(TopicReplyBestAnswer, {
      props: { reply: reply({ is_best_answer: false }) },
      global: { provide }
    })
    await wrapper.find('button').trigger('click')
    expect(setting[0]!.method).toBe('PUT')
    wrapper.unmount()
  })

  it('unpins with DELETE and pins with PUT', async () => {
    const provide = {
      pageTopic: computed(() =>
        topic({ viewer: topicViewer({ can_pin_reply: true }) })
      ),
      replaceTopic: () => undefined
    }
    const captured = captureFetch(topic())
    const unpinning = await mountSuspended(TopicReplyPin, {
      props: { reply: reply({ is_pinned: true }) },
      global: { provide }
    })
    await unpinning.find('button').trigger('click')
    expect(captured[0]!.method).toBe('DELETE')
    expect(captured[0]!.url).toContain('/topics/42/pinned-reply')
    unpinning.unmount()

    const pinning = captureFetch(topic())
    const wrapper = await mountSuspended(TopicReplyPin, {
      props: { reply: reply({ is_pinned: false }) },
      global: { provide }
    })
    await wrapper.find('button').trigger('click')
    expect(pinning[0]!.method).toBe('PUT')
    wrapper.unmount()
  })
})
