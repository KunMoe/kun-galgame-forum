// @vitest-environment nuxt
import { afterEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { ContentDocument } from '#shared/utils/api/schemas'
import { KUN_GALGAME_RATING_SPOILER_WARNING } from '~/constants/galgame-rating'
import Card from './Card.vue'

let wrapper: VueWrapper | undefined

const documentOf = (summary: string): ContentDocument => ({
  object: 'document',
  children: summary
    ? [
        {
          object: 'paragraph',
          children: summary.split(/\|\|(.*?)\|\|/gs).map((value, index) =>
            index % 2
              ? {
                  object: 'inline_spoiler',
                  children: [{ object: 'text', value }]
                }
              : { object: 'text', value }
          )
        }
      ]
    : []
})

const mountRating = async (
  spoilerLevel: string,
  summary: string,
  summaryContent: ContentDocument | null = documentOf(summary)
) => {
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
          summary_content: summaryContent as ContentDocument,
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

  it('renders a legacy API response without summary_content', async () => {
    const card = await mountRating('portion', '音乐很好，||秘密||，推荐', null)

    expect(card.text()).toContain('音乐很好，')
    expect(card.text()).not.toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
    expect(card.findAll('.kun-spoiler-hidden')).toHaveLength(1)
  })

  it('shows plain text without Markdown formatting or link destinations', async () => {
    const card = await mountRating(
      'none',
      '**音乐** [剧情](https://example.com) `||code||`',
      {
        object: 'document',
        children: [
          {
            object: 'paragraph',
            children: [
              {
                object: 'strong',
                children: [{ object: 'text', value: '音乐' }]
              },
              { object: 'text', value: ' ' },
              {
                object: 'link',
                url: 'https://example.com',
                children: [{ object: 'text', value: '剧情' }]
              },
              { object: 'text', value: ' ' },
              { object: 'inline_code', value: '||code||' }
            ]
          }
        ]
      }
    )

    expect(card.text()).toContain('音乐 剧情 ||code||')
    expect(card.text()).not.toContain('**')
    expect(card.text()).not.toContain('https://example.com')
    expect(card.find('.kun-spoiler').exists()).toBe(false)
  })

  it('does not treat pipe characters in code as a partial spoiler mask', async () => {
    const card = await mountRating('portion', '`||结局||`', {
      object: 'document',
      children: [
        {
          object: 'paragraph',
          children: [{ object: 'inline_code', value: '||结局||' }]
        }
      ]
    })

    expect(card.text()).toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
    expect(card.text()).not.toContain('结局')
  })

  it('strips formatting inside a real partial spoiler while keeping it masked', async () => {
    const card = await mountRating(
      'portion',
      '音乐很好，||**真结局**||，值得玩',
      {
        object: 'document',
        children: [
          {
            object: 'paragraph',
            children: [
              { object: 'text', value: '音乐很好，' },
              {
                object: 'inline_spoiler',
                children: [
                  {
                    object: 'strong',
                    children: [{ object: 'text', value: '真结局' }]
                  }
                ]
              },
              { object: 'text', value: '，值得玩' }
            ]
          }
        ]
      }
    )

    expect(card.text()).toContain('音乐很好，')
    expect(card.text()).toContain('，值得玩')
    expect(card.text()).not.toContain('**')
    expect(card.text()).not.toContain(KUN_GALGAME_RATING_SPOILER_WARNING)
    expect(card.findAll('.kun-spoiler-hidden')).toHaveLength(1)
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
