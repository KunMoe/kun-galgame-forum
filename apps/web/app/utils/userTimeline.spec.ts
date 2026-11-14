import { describe, expect, it } from 'vitest'
import type { UserActivityGroup } from '#shared/utils/api/schemas'
import { groupTimelineByDay, timelineDayLabel } from './userTimeline'

const group = (id: string, date: string) =>
  ({ id, calendar_date: date }) as UserActivityGroup

describe('timelineDayLabel', () => {
  it('names the weekday from the calendar date alone', () => {
    expect(timelineDayLabel('2026-11-14', '2026')).toBe('11 月 14 日 周六')
  })

  it('adds the year only when it differs from the newest group', () => {
    expect(timelineDayLabel('2025-01-05', '2026')).toBe(
      '2025 年 1 月 5 日 周日'
    )
  })
})

describe('groupTimelineByDay', () => {
  it('keeps the feed order and starts a new day on every date change', () => {
    const days = groupTimelineByDay([
      group('1', '2026-11-14'),
      group('2', '2026-11-14'),
      group('3', '2026-11-12'),
      group('4', '2026-11-14')
    ])
    expect(days.map((d) => [d.date, d.groups.map((g) => g.id)])).toEqual([
      ['2026-11-14', ['1', '2']],
      ['2026-11-12', ['3']],
      ['2026-11-14', ['4']]
    ])
  })
})
