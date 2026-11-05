import { describe, expect, it } from 'vitest'
import { isRatingSummaryShown } from './ratingCard'

describe('isRatingSummaryShown', () => {
  it.each([
    ['none', false, true],
    ['none', true, true],
    ['portion', true, true],
    ['portion', false, false],
    ['serious', true, false],
    ['serious', false, false]
  ])('%s with mask %s shows: %s', (level, mask, want) => {
    expect(
      isRatingSummaryShown({
        spoiler_level: level,
        has_short_summary_spoiler: mask
      })
    ).toBe(want)
  })
})
