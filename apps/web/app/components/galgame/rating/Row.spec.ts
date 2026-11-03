// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import Row from './Row.vue'

describe('rating row excerpts', () => {
  it('masks a long inline spoiler before truncating the legacy review', async () => {
    const wrapper = await mountSuspended(Row, {
      props: {
        rating: {
          id: 31,
          user: { id: 7, name: 'rater', avatar: '' },
          overall: 8,
          recommend: 'yes',
          spoiler_level: 'none',
          short_summary: `音乐很好，||${'秘密'.repeat(100)}||，推荐`
        } as GalgameRatingCardOnGalgamePage
      }
    })

    expect(wrapper.text()).toContain('音乐很好，███，推荐')
    expect(wrapper.text()).not.toContain('秘密')
    wrapper.unmount()
  })
})
