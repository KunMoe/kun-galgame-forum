// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import Row from './Row.vue'

describe('rating row previews', () => {
  it('masks a spoiler before the row truncates the review', async () => {
    const wrapper = await mountSuspended(Row, {
      props: {
        rating: {
          id: 31,
          user: { id: 7, name: 'rater', avatar: '' },
          overall: 8,
          recommend: 'yes',
          spoiler_level: 'none',
          short_summary: `音乐很棒||${'秘密'.repeat(100)}||值得一玩`
        } as GalgameRatingCardOnGalgamePage
      }
    })

    expect(wrapper.text()).toContain('音乐很棒███值得一玩')
    expect(wrapper.text()).not.toContain('秘密')
    expect(wrapper.get('a').attributes('href')).toBe('/galgame-rating/31')
    wrapper.unmount()
  })
})
