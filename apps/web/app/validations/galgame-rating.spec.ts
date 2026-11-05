import { describe, expect, it } from 'vitest'
import { galgameRatingFormSchema } from './galgame-rating'

const form = (overall: number, short_summary: string) => ({
  recommend: 'neutral',
  overall,
  game_types: ['plot'],
  spoiler_level: 'none',
  short_summary,
  aspect_scores: {
    art: null,
    story: null,
    music: null,
    character: null,
    route: null,
    system: null,
    voice: null,
    replay_value: null
  }
})

describe('galgameRatingFormSchema', () => {
  it('counts the 2000 limit in characters, as NextMoe does', () => {
    expect(
      galgameRatingFormSchema.safeParse(form(8, '🎮'.repeat(2000))).success
    ).toBe(true)
    expect(
      galgameRatingFormSchema.safeParse(form(8, '🎮'.repeat(2001))).success
    ).toBe(false)
  })

  it('measures the 100-character rule on the text, not the markup', () => {
    const padded = `**${'好'.repeat(97)}**`
    expect(galgameRatingFormSchema.safeParse(form(10, padded)).success).toBe(
      false
    )
    expect(
      galgameRatingFormSchema.safeParse(form(10, '好'.repeat(100))).success
    ).toBe(true)
  })
})
