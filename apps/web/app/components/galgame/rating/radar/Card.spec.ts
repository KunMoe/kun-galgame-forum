// @vitest-environment nuxt
import { afterEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import { KUN_GALGAME_RATING_SPOILER_WARNING } from '~/constants/galgame-rating'
import Card from './Card.vue'

let wrapper: VueWrapper | undefined

const mountRating = async (spoilerLevel: string, summary: string) => {
  wrapper = await mountSuspended(Card, {
    props: {
      ratings: [
        {
          id: 31,
          user: { id: 7, name: 'rater', avatar: '' },
          overall: 8,
          recommend: 'yes',
          play_status: 'done_all',
          spoiler_level: spoilerLevel,
          short_summary: summary,
          art: 8,
          story: 8,
          music: 8,
          character: 8,
          route: 8,
          system: 8,
          voice: 8,
          replay_value: 8,
          galgame_type: ['plot'],
          view: 0,
          like_count: 0,
          is_liked: false,
          created: '2026-10-30T10:00:00Z',
          updated: '2026-10-30T10:00:00Z',
          galgame: { id: 228620, name: 'Game', content_limit: 'sfw' }
        }
      ]
    }
  })
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('radar rating card summaries', () => {
  it('shows a review with no spoilers', async () => {
    const card = await mountRating('none', '音乐很棒，值得一玩')

    expect(card.text()).toContain('音乐很棒，值得一玩')
    expect(card.text()).not.toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
  })

  it.each(['none', 'portion'])(
    'shows surrounding text and reveals only the clicked passage (%s)',
    async (level) => {
      const card = await mountRating(
        level,
        '音乐很棒，||秘密一||，||秘密第一行\n第二行||，值得一玩'
      )

      expect(card.text()).toContain('音乐很棒，')
      expect(card.text()).toContain('，值得一玩')
      expect(card.text()).not.toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
      const spoilers = card.findAll('.kun-spoiler-hidden')
      expect(spoilers).toHaveLength(2)
      expect(spoilers[1]!.text()).toBe('秘密第一行\n第二行')

      await spoilers[0]!.trigger('click')
      expect(spoilers[0]!.attributes('aria-expanded')).toBe('true')
      expect(spoilers[1]!.classes()).toContain('kun-spoiler-hidden')
    }
  )

  it.each([
    ['portion', '音乐很棒，但结局包含秘密'],
    ['portion', '音乐很棒，但||秘密未闭合'],
    ['serious', '音乐很棒，但结局包含秘密'],
    ['serious', '音乐很棒，但||秘密||']
  ] as const)('shows only a warning for %s: %s', async (level, summary) => {
    const card = await mountRating(level, summary)

    expect(card.text()).toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
    expect(card.text()).not.toContain('音乐很棒')
    expect(card.text()).not.toContain('秘密')
    expect(card.find('.kun-spoiler').exists()).toBe(false)
    expect(card.get('a[href="/galgame-rating/31"]').text()).toContain(
      '阅读详情'
    )
  })

  it.each(['none', 'portion', 'serious'])(
    'does not show a warning for an empty summary (%s)',
    async (level) => {
      const card = await mountRating(level, '')

      expect(card.text()).not.toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
      expect(card.find('.kun-spoiler').exists()).toBe(false)
    }
  )
})
