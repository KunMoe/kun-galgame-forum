// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { FollowingActivityItem } from '#shared/utils/api/schemas'
import FollowingItem from './FollowingItem.vue'

const name = (display_name: string) => ({
  display_name,
  latin: null,
  localized: {}
})

const image = {
  hash: 'x',
  sexual: null,
  url: 'https://image.test.example/ab/cd/x.webp',
  width: 600,
  height: 800,
  thumbhash: null
}

const base: FollowingActivityItem = {
  object: 'following_activity_item',
  id: '1',
  site: 'kungal',
  key: 'k1',
  verb: 'rate',
  object_kind: 'galgame_rating',
  object_label: '评分',
  title: 'Work 77',
  excerpt: '',
  url: 'https://www.kungal.com/galgame-rating/5',
  in_site_path: '/galgame-rating/5',
  cover: null,
  related_work_id: null,
  work: null,
  is_nsfw: false,
  occurred_at: '2026-11-14T08:00:00Z',
  topic_times: null
}

describe('HomeFollowingItem', () => {
  it('shows the maker, release date and tags of the work, with its cover', async () => {
    const wrapper = await mountSuspended(FollowingItem, {
      props: {
        item: {
          ...base,
          related_work_id: '77',
          work: {
            object: 'work',
            id: '77',
            ...name('Work 77'),
            cover: image,
            cover_sfw: image,
            is_nsfw: false,
            maker: { object: 'company', id: '7', ...name('Brand') },
            top_tags: [
              { object: 'tag', id: '11', ...name('Island'), is_sexual: false }
            ],
            release_date: '2024-05-01',
            release_date_precision: 'month'
          }
        }
      }
    })
    const text = wrapper.text()
    expect(text).toContain('Brand')
    expect(text).toContain('2024-05')
    expect(text).toContain('Island')
    expect(wrapper.find('a[href="/galgame/official/7"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/galgame/tag/11"]').exists()).toBe(true)
    expect(wrapper.find('img').exists()).toBe(true)
  })

  it('stays a plain row for an item without a work', async () => {
    const wrapper = await mountSuspended(FollowingItem, {
      props: { item: { ...base, title: 'Topic' } }
    })
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('a[href^="/galgame/tag/"]').exists()).toBe(false)
  })
})
