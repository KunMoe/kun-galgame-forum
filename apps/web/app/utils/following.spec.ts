import { describe, expect, it } from 'vitest'
import { followingGroupPhrase, followingSiteLabel } from './following'

describe('followingGroupPhrase', () => {
  it('counts only when the group holds more than one item', () => {
    expect(
      followingGroupPhrase({
        verb: 'publish',
        item_count: 1,
        object_kind: 'topic',
        object_label: '话题'
      })
    ).toBe('发布了话题')
    expect(
      followingGroupPhrase({
        verb: 'publish',
        item_count: 3,
        object_kind: 'topic',
        object_label: '话题'
      })
    ).toBe('发布了 3 个话题')
  })

  it('spaces a label that starts with a Latin letter', () => {
    expect(
      followingGroupPhrase({
        verb: 'publish',
        item_count: 2777,
        object_kind: 'topic',
        object_label: 'Galgame 资源'
      })
    ).toBe('发布了 2777 个 Galgame 资源')
    expect(
      followingGroupPhrase({
        verb: 'edit',
        item_count: 1,
        object_kind: 'topic',
        object_label: 'Galgame'
      })
    ).toBe('编辑了 Galgame')
  })

  it('uses 条 for replies, comments and ratings', () => {
    expect(
      followingGroupPhrase({
        verb: 'reply',
        item_count: 5,
        object_kind: 'topic',
        object_label: '回复'
      })
    ).toBe('发表了 5 条回复')
    expect(
      followingGroupPhrase({
        verb: 'rate',
        item_count: 2,
        object_kind: 'topic',
        object_label: '评分'
      })
    ).toBe('发表了 2 条评分')
  })

  it('words a play-state mark by the state it set', () => {
    expect(
      followingGroupPhrase({
        verb: 'mark',
        item_count: 3,
        object_kind: 'work_wish',
        object_label: '作品'
      })
    ).toBe('想玩 3 个作品')
    expect(
      followingGroupPhrase({
        verb: 'mark',
        item_count: 1,
        object_kind: 'work_someday',
        object_label: '作品'
      })
    ).toBe('标记了作品')
  })
})

describe('followingSiteLabel', () => {
  it('hides this forum and names the others', () => {
    expect(followingSiteLabel('kungal')).toBe('')
    expect(followingSiteLabel('catalog')).toBe('')
    expect(followingSiteLabel('moyu')).toBe('moyu.moe')
    expect(followingSiteLabel('somewhere')).toBe('somewhere')
  })
})
