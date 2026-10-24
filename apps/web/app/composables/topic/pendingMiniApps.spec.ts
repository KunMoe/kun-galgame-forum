// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import type { PollFormData } from '~/components/topic/poll/types'
import type { LotteryFormData } from '~/components/topic/lottery/types'
import {
  lotteryFormFromDraft,
  lotteryFormToDraft,
  lotteryMissesCodes,
  lotteryPointBudget,
  lotteryWithoutCodes,
  pendingLotteryIssue,
  pendingPollIssue,
  pollFormFromDraft,
  pollFormToDraft
} from './pendingMiniApps'

const HASH = 'ab'.repeat(32)

const poll = (over: Partial<PollFormData> = {}): PollFormData => ({
  title: 'Route',
  description: '',
  options: [{ text: 'a' }, { text: 'b', _status: 'deleted' }, { text: 'c' }],
  choice_type: 'multiple',
  min_choice: 1,
  max_choice: 2,
  closes_at: '2099-12-31',
  result_visibility: 'after_vote',
  is_anonymous: true,
  can_change_vote: false,
  ...over
})

const lottery = (over: Partial<LotteryFormData> = {}): LotteryFormData => ({
  title: 'Keys',
  description: '',
  entry_mode: 'signup',
  floor_rule: '8,18',
  draw_mode: 'deadline',
  draw_threshold: 0,
  deadline: '2099-12-31',
  min_account_age_days: 3,
  min_moemoepoint: 20,
  show_entrants: true,
  prizes: [
    {
      name: 'Steam key',
      description: '',
      image_hashes: [HASH],
      image_urls: ['https://cdn.example/x.webp'],
      nsfw_hashes: [HASH, 'cd'.repeat(32)],
      machine_nsfw_hashes: [],
      delivery: 'code',
      point_mode: 'fixed',
      point_amount: 0,
      slots: 2,
      codes: 'AAAA\nBBBB\n'
    },
    {
      name: 'Points',
      description: '',
      image_hashes: [],
      image_urls: [],
      nsfw_hashes: [],
      machine_nsfw_hashes: [],
      delivery: 'point',
      point_mode: 'fixed',
      point_amount: 30,
      slots: 3,
      codes: ''
    }
  ],
  ...over
})

describe('pending mini-apps', () => {
  it('round-trips a poll through the draft shape, dropping deleted options', () => {
    const draft = pollFormToDraft(poll())
    expect(draft.options).toEqual([{ text: 'a' }, { text: 'c' }])
    expect(draft.closes_at).toMatch(/^2099-12-31T|^2100-01-01T/)
    expect(pollFormFromDraft(draft)).toMatchObject({
      title: 'Route',
      closes_at: '2099-12-31',
      options: [{ text: 'a' }, { text: 'c' }]
    })
  })

  it('never puts codes into the draft shape and restores everything else', () => {
    const draft = lotteryFormToDraft(lottery())
    expect(JSON.stringify(draft)).not.toContain('AAAA')
    expect(draft.floor_rule).toBeNull()
    expect(draft.prizes[0]!.adult_image_hashes).toEqual([HASH])
    expect(draft.prizes[0]!.point_mode).toBeNull()
    expect(draft.prizes[1]).toMatchObject({ point_mode: 'fixed', point_amount: 30 })

    const back = lotteryFormFromDraft(draft)
    expect(back.deadline).toBe('2099-12-31')
    expect(back.prizes[0]!.codes).toBe('')
    expect(back.prizes[0]!.image_urls[0]).toContain(HASH)
    expect(lotteryMissesCodes(back)).toBe(true)
  })

  it('strips codes for local storage without touching the form', () => {
    const form = lottery()
    expect(lotteryWithoutCodes(form).prizes[0]!.codes).toBe('')
    expect(form.prizes[0]!.codes).toBe('AAAA\nBBBB\n')
  })

  it('sums the point budget of point prizes only', () => {
    expect(lotteryPointBudget(lottery())).toBe(90)
  })

  it('refuses what createPoll or createLottery would refuse after the topic exists', () => {
    expect(pendingPollIssue(poll())).toBe('')
    expect(pendingPollIssue(poll({ closes_at: '2001-01-01' }))).not.toBe('')
    expect(pendingLotteryIssue(lottery())).toBe('')
    expect(pendingLotteryIssue(lottery({ deadline: undefined }))).not.toBe('')
    expect(pendingLotteryIssue(lottery({ deadline: '2001-01-01' }))).not.toBe('')
    expect(pendingLotteryIssue(lotteryWithoutCodes(lottery()))).not.toBe('')
  })
})
