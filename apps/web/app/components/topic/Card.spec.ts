// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { TopicSummary, UserRef } from '#shared/utils/api/schemas'
import TopicCard from './Card.vue'

const user = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '3',
  name: 'Alice',
  avatar: {
    hash: 'ab',
    url: 'https://cdn.example/ab.webp',
    width: 64,
    height: 64,
    thumbhash: null,
    sexual: null
  },
  ...over
})

const topic = (over: Partial<TopicSummary> = {}): TopicSummary => ({
  object: 'topic',
  id: '42',
  title: 'A topic',
  state: 'published',
  category: 'galgame',
  sections: ['g-chatting'],
  cover_images: [],
  author: user(),
  view_count: 12345,
  like_count: 0,
  reply_count: 1,
  comment_count: 2,
  has_best_answer: false,
  mini_apps: [],
  is_nsfw: false,
  bumped_at: '2026-09-01T00:00:00.000Z',
  created_at: '2026-08-01T00:00:00.000Z',
  edited_at: null,
  upvoted_at: null,
  ...over
})

describe('TopicCard', () => {
  it('renders a deleted author as 已注销用户, the formatted view count, and section badges', async () => {
    const wrapper = await mountSuspended(TopicCard, {
      props: {
        topic: topic({ author: user({ name: null, avatar: null }) })
      }
    })
    expect(wrapper.text()).toContain('已注销用户')
    expect(wrapper.text()).toContain('1.2w')
    expect(wrapper.text()).toContain('闲聊')
  })

  it('shows the bump and edit times, not the creation time', async () => {
    const wrapper = await mountSuspended(TopicCard, {
      props: { topic: topic({ edited_at: '2026-08-15T00:00:00.000Z' }) }
    })
    const times = wrapper.findAll('time').map((t) => t.attributes('datetime'))
    expect(times).toEqual([
      '2026-09-01T00:00:00.000Z',
      '2026-08-15T00:00:00.000Z'
    ])
  })
})
