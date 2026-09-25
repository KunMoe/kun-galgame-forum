// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import type { Notification, NotificationType } from '#shared/utils/api/schemas'
import { getMessageI18n } from './getMessageI18n'

const ALL_TYPES = {
  upvoted: true,
  liked: true,
  favorited: true,
  replied: true,
  commented: true,
  mentioned: true,
  followed_thread_activity: true,
  best_answer_chosen: true,
  reply_pinned: true,
  quiz_answered: true,
  resource_link_reported: true,
  edit_requested: true,
  edit_merged: true,
  edit_declined: true,
  lottery_won: true,
  lottery_drawn: true,
  lottery_code_expired: true,
  poll_closed: true,
  user_followed: true,
  followee_topic_created: true
} as const satisfies Record<NotificationType, true>

const notification = (over: Partial<Notification> = {}): Notification => ({
  object: 'notification',
  id: '1',
  notification_type: 'liked',
  actor: { object: 'user', id: '2', name: 'A', avatar: null },
  actor_count: 1,
  item_count: 1,
  path: '/topic/1',
  excerpt_markdown: '',
  origin: 'local',
  is_read: false,
  created_at: '2026-01-01T00:00:00Z',
  ...over
})

describe('getMessageI18n', () => {
  it('has a template for every v1 token', () => {
    for (const notification_type of Object.keys(
      ALL_TYPES
    ) as NotificationType[]) {
      expect(getMessageI18n(notification({ notification_type }))).not.toBe('')
    }
  })

  it('uses the replied template for a local mentioned row with excerpt', () => {
    expect(
      getMessageI18n(
        notification({
          notification_type: 'mentioned',
          excerpt_markdown: 'hello',
          origin: 'local'
        })
      )
    ).toBe(' 回复了您!')
  })

  it('keeps the mentioned template when the excerpt is empty or the source is community', () => {
    expect(
      getMessageI18n(
        notification({
          notification_type: 'mentioned',
          excerpt_markdown: '',
          origin: 'local'
        })
      )
    ).toBe(' 提到了您！')
    expect(
      getMessageI18n(
        notification({
          notification_type: 'mentioned',
          excerpt_markdown: 'hello',
          origin: 'community'
        })
      )
    ).toBe(' 提到了您！')
  })

  it('folds liked by actor_count', () => {
    expect(
      getMessageI18n(
        notification({ notification_type: 'liked', actor_count: 3 })
      )
    ).toBe(' 等 3 人点赞了您!')
  })

  it('folds user_followed by actor_count', () => {
    expect(
      getMessageI18n(notification({ notification_type: 'user_followed' }))
    ).toBe(' 关注了您!')
    expect(
      getMessageI18n(
        notification({ notification_type: 'user_followed', actor_count: 3 })
      )
    ).toBe(' 等 3 人关注了您!')
  })

  it('folds followed_thread_activity by actor_count and item_count', () => {
    expect(
      getMessageI18n(
        notification({
          notification_type: 'followed_thread_activity',
          actor_count: 4,
          item_count: 6
        })
      )
    ).toBe(' 等 4 人在您关注的评论区发表了 6 条新评论')
    expect(
      getMessageI18n(
        notification({
          notification_type: 'followed_thread_activity',
          actor_count: 1,
          item_count: 5
        })
      )
    ).toBe(' 在您关注的评论区发表了 5 条新评论')
  })
})
