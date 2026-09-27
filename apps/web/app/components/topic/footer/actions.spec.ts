// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { Topic, TopicViewer, UserRef } from '#shared/utils/api/schemas'
import TopicFooterRewrite from './Rewrite.vue'
import TopicFooterHide from './Hide.vue'
import TopicFooterUpvote from './Upvote.vue'

const user = (): UserRef => ({
  object: 'user',
  id: '1',
  name: 'Alice',
  avatar: null,
  avatar_frame: null
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
  favorite_count: 0,
  hidden_by: null,
  is_nsfw: false,
  like_count: 0,
  mini_apps: [],
  pinned_reply: null,
  reactions: [],
  reply_count: 0,
  sections: ['g-chatting'],
  state: 'published',
  upvote_count: 0,
  upvoted_at: null,
  view_count: 1,
  viewer: topicViewer(),
  ...over
})

describe('topic footer buttons follow viewer.can_*', () => {
  it('hides rewrite when can_edit is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicFooterRewrite, {
      props: {
        menu: true,
        topic: topic({ viewer: topicViewer({ can_edit: false }) })
      }
    })
    expect(hidden.text()).not.toContain('重新编辑')
    hidden.unmount()
    const shown = await mountSuspended(TopicFooterRewrite, {
      props: {
        menu: true,
        topic: topic({ viewer: topicViewer({ can_edit: true }) })
      }
    })
    expect(shown.text()).toContain('重新编辑')
    shown.unmount()
  })

  it('hides hide when can_hide is false and shows unhide when can_unhide is true', async () => {
    const hidden = await mountSuspended(TopicFooterHide, {
      props: { topic: topic({ viewer: topicViewer({ can_hide: false }) }) }
    })
    expect(hidden.text()).not.toContain('隐藏该话题')
    hidden.unmount()
    const shown = await mountSuspended(TopicFooterHide, {
      props: { topic: topic({ viewer: topicViewer({ can_hide: true }) }) }
    })
    expect(shown.text()).toContain('隐藏该话题')
    shown.unmount()
    const unhide = await mountSuspended(TopicFooterHide, {
      props: {
        topic: topic({
          state: 'hidden',
          viewer: topicViewer({ can_unhide: true })
        })
      }
    })
    expect(unhide.text()).toContain('取消隐藏该话题')
    unhide.unmount()
  })

  it('hides upvote when can_upvote is false and shows it when true', async () => {
    const hidden = await mountSuspended(TopicFooterUpvote, {
      props: {
        menu: true,
        topic: topic({ viewer: topicViewer({ can_upvote: false }) })
      }
    })
    expect(hidden.text()).not.toContain('推话题')
    hidden.unmount()
    const shown = await mountSuspended(TopicFooterUpvote, {
      props: {
        menu: true,
        topic: topic({ viewer: topicViewer({ can_upvote: true }) })
      }
    })
    expect(shown.text()).toContain('推话题')
    shown.unmount()
  })
})
