// @vitest-environment nuxt
import { afterEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type {
  Activity,
  ActivityRating,
  WorkRef
} from '#shared/utils/api/schemas'
import GalgameRating from './GalgameRating.vue'

let wrapper: VueWrapper | undefined

afterEach(() => wrapper?.unmount())

const work: WorkRef = {
  object: 'work',
  id: '228620',
  display_name: 'Game',
  latin: null,
  localized: {},
  cover: null,
  is_nsfw: false
}
const activity = {
  path: '/galgame-rating/31',
  occurred_at: '2026-10-30T10:00:00Z',
  performer: null
} as Activity

const mountRating = async (
  summary: string,
  spoilerLevel: ActivityRating['spoiler_level'] = 'none'
) => {
  wrapper = await mountSuspended(GalgameRating, {
    props: {
      activity,
      work,
      rating: {
        rating_id: '31',
        overall: 8,
        recommend: 'yes',
        play_status: null,
        spoiler_level: spoilerLevel,
        short_summary: summary,
        like_count: 0
      }
    },
    global: {
      stubs: {
        ActivityCardShell: { template: '<div><slot /></div>' },
        ActivityCardWorkCover: true,
        GalgameRatingDetailLike: true
      }
    }
  })
  return wrapper
}

describe('rating activity summaries', () => {
  it('keeps long inline spoilers clickable and links to the complete review', async () => {
    const secret = '结局真相'.repeat(150)
    const card = await mountRating(`前文||${secret}||后文`)

    const spoiler = card.get('.kun-spoiler-hidden')
    expect(spoiler.text()).toBe(secret)
    expect(card.text()).toContain('前文')
    await spoiler.trigger('click')
    expect(card.find('.kun-spoiler-hidden').exists()).toBe(false)
    expect(card.get('a[href="/galgame-rating/31"]').text()).toContain(
      '查看详情'
    )
  })

  it.each(['portion', 'serious'] as const)(
    'continues to withhold the declared %s summary',
    async (level) => {
      const card = await mountRating('前文||结局真相||后文', level)

      expect(card.text()).toContain('该评分可能含有剧透内容，点进查看')
      expect(card.text()).not.toContain('结局真相')
      expect(card.find('.kun-spoiler').exists()).toBe(false)
    }
  )
})
