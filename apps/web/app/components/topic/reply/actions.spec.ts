// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { computed } from 'vue'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type {
  Reply,
  ReplyViewer,
  Topic,
  TopicViewer,
  UserRef
} from '#shared/utils/api/schemas'
import TopicReplyDelete from './Delete.vue'
import TopicReplyBestAnswer from './BestAnswer.vue'
import TopicReplyPin from './Pin.vue'
import TopicReplyRewrite from './Rewrite.vue'

const user = (): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null,
  avatar_frame: null
})

const replyViewer = (over: Partial<ReplyViewer> = {}): ReplyViewer => ({
  can_edit: false,
  can_delete: false,
  can_like: false,
  has_liked: false,
  has_disliked: false,
  ...over
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
  favorite_count: 0,
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

describe('reply action buttons follow viewer.can_*', () => {
  it('hides delete when can_delete is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicReplyDelete, {
      props: { reply: reply({ viewer: replyViewer({ can_delete: false }) }) }
    })
    expect(hidden.text()).not.toContain('删除回复')
    hidden.unmount()
    const shown = await mountSuspended(TopicReplyDelete, {
      props: { reply: reply({ viewer: replyViewer({ can_delete: true }) }) }
    })
    expect(shown.text()).toContain('删除回复')
    shown.unmount()
  })

  it('hides rewrite when can_edit is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicReplyRewrite, {
      props: { reply: reply({ viewer: replyViewer({ can_edit: false }) }) }
    })
    expect(hidden.text()).not.toContain('重新编辑')
    hidden.unmount()
    const shown = await mountSuspended(TopicReplyRewrite, {
      props: { reply: reply({ viewer: replyViewer({ can_edit: true }) }) }
    })
    expect(shown.text()).toContain('重新编辑')
    shown.unmount()
  })

  it('hides best-answer when can_set_best_answer is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicReplyBestAnswer, {
      props: { reply: reply() },
      global: {
        provide: {
          pageTopic: computed(() =>
            topic({ viewer: topicViewer({ can_set_best_answer: false }) })
          ),
          replaceTopic: () => undefined
        }
      }
    })
    expect(hidden.text()).not.toContain('最佳答案')
    hidden.unmount()
    const shown = await mountSuspended(TopicReplyBestAnswer, {
      props: { reply: reply() },
      global: {
        provide: {
          pageTopic: computed(() =>
            topic({ viewer: topicViewer({ can_set_best_answer: true }) })
          ),
          replaceTopic: () => undefined
        }
      }
    })
    expect(shown.text()).toContain('将该回复设为最佳答案')
    shown.unmount()
  })

  it('hides pin when can_pin_reply is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicReplyPin, {
      props: { reply: reply() },
      global: {
        provide: {
          pageTopic: computed(() =>
            topic({ viewer: topicViewer({ can_pin_reply: false }) })
          ),
          replaceTopic: () => undefined
        }
      }
    })
    expect(hidden.text()).not.toContain('置顶回复')
    hidden.unmount()
    const shown = await mountSuspended(TopicReplyPin, {
      props: { reply: reply() },
      global: {
        provide: {
          pageTopic: computed(() =>
            topic({ viewer: topicViewer({ can_pin_reply: true }) })
          ),
          replaceTopic: () => undefined
        }
      }
    })
    expect(shown.text()).toContain('置顶回复')
    shown.unmount()
  })
})
