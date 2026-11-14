import { describe, expect, it } from 'vitest'
import type { UserProfile } from '#shared/utils/api/schemas'
import {
  findUserNavSection,
  userProfileNav,
  userSectionTabs
} from './userProfileNav'

const user = (counts: Partial<UserProfile['counts']> = {}) =>
  ({
    id: '7',
    counts: {
      topic_count: 3,
      reply_count: 0,
      topic_comment_count: 0,
      published_galgame_count: 0,
      galgame_rating_count: 0,
      galgame_resource_count: 0,
      toolset_count: 0,
      following_count: 2,
      follower_count: 5,
      ...counts
    }
  }) as UserProfile

const sectionValues = (nav: ReturnType<typeof userProfileNav>, item: string) =>
  nav
    .find((i) => i.value === item)!
    .groups.flatMap((g) => g.sections.map((s) => s.value))

describe('userProfileNav', () => {
  it('hides only the sections a count fully covers, and only from visitors', () => {
    const visitor = userProfileNav(user(), false)
    expect(sectionValues(visitor, 'publish')).toEqual([
      'topic',
      'reply',
      'comment',
      'galgame',
      'quiz',
      'resource'
    ])
    const owner = userProfileNav(user(), true)
    expect(sectionValues(owner, 'publish')).toContain('rating')
    expect(sectionValues(owner, 'publish')).toContain('toolset')
  })

  it('keeps the owner-only sections and the settings tab to the owner', () => {
    const visitor = userProfileNav(user(), false)
    expect(sectionValues(visitor, 'library')).toEqual(['library'])
    expect(sectionValues(visitor, 'follow')).toEqual(['following', 'followers'])
    expect(visitor.map((i) => i.value)).not.toContain('setting')

    const owner = userProfileNav(user(), true)
    expect(sectionValues(owner, 'library')).toEqual(['library', 'playtime'])
    expect(sectionValues(owner, 'follow')).toContain('followed-companies')
    expect(owner.map((i) => i.value)).toContain('setting')
  })

  it('resolves sub-pages to their section without prefix collisions', () => {
    const nav = userProfileNav(user(), true)
    const at = (path: string) => {
      const hit = findUserNavSection(nav, path)
      return hit && `${hit.item.value}/${hit.section.value}`
    }
    expect(at('/user/7/topic/topic-like')).toBe('publish/topic')
    expect(at('/user/7/follow/followers')).toBe('follow/followers')
    expect(at('/user/7/followed-companies')).toBe('follow/followed-companies')
    expect(at('/user/7/collection/toolset')).toBe(
      'collection/collection-toolset'
    )
    expect(at('/user/7/galgame/galgame-like')).toBe('publish/galgame')
    expect(at('/user/7/galgame-rating')).toBeNull()
  })

  it('labels a section with its count when one is known', () => {
    const nav = userProfileNav(user(), false)
    const discussion = nav.find((i) => i.value === 'publish')!.groups[0]!
    expect(userSectionTabs(discussion).map((t) => t.textValue)).toEqual([
      '话题 3',
      '回复 0',
      '评论 0'
    ])
  })
})
