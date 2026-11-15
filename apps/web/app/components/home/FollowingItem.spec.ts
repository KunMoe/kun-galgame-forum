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
  work_preview: null,
  object_ref: null,
  object_target: null,
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
          work_preview: {
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

  it('names a followed user from the object target, not the empty title', async () => {
    const wrapper = await mountSuspended(FollowingItem, {
      props: {
        item: {
          ...base,
          site: 'moyu',
          verb: 'follow',
          object_kind: 'user',
          object_label: '用户',
          title: '',
          url: 'https://www.kungal.com/user/9',
          in_site_path: '/user/9',
          object_ref: 'user:9',
          object_target: {
            object: 'user',
            id: '9',
            name: 'Nine',
            avatar: null,
            avatar_frame: null
          }
        }
      }
    })
    expect(wrapper.text()).toContain('Nine')
    expect(wrapper.find('a[href="/user/9"]').exists()).toBe(true)
    expect(wrapper.find('a[target="_blank"]').exists()).toBe(false)
  })

  it('names a marked work from its preview', async () => {
    const wrapper = await mountSuspended(FollowingItem, {
      props: {
        item: {
          ...base,
          site: 'catalog',
          verb: 'mark',
          object_kind: 'work_done',
          object_label: '作品',
          title: 'Old name',
          url: 'https://www.kungal.com/galgame/77',
          in_site_path: '/galgame/77',
          related_work_id: '77',
          object_ref: 'work:77',
          work_preview: {
            object: 'work',
            id: '77',
            ...name('New name'),
            cover: null,
            cover_sfw: null,
            is_nsfw: false,
            maker: null,
            top_tags: [],
            release_date: null,
            release_date_precision: null
          }
        }
      }
    })
    expect(wrapper.text()).toContain('New name')
    expect(wrapper.text()).not.toContain('Old name')
  })

  it('stays a plain row for an item without a work', async () => {
    const wrapper = await mountSuspended(FollowingItem, {
      props: { item: { ...base, title: 'Topic' } }
    })
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('a[href^="/galgame/tag/"]').exists()).toBe(false)
  })
})
