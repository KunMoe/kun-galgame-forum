import type {
  PollCreate,
  TopicDraftLottery,
  TopicDraftPoll
} from '#shared/utils/api/schemas'
import type { PollFormData } from '~/components/topic/poll/types'
import type {
  LotteryFormData,
  LotteryPrizeFormData
} from '~/components/topic/lottery/types'
import {
  closesAtFromPicker,
  deadlineFromPicker,
  deadlineToPicker
} from '~/components/topic/miniapp/deadline'
import { imageTokenUrl } from '~/utils/imageSrc'

export const clonePlain = <T>(value: T): T =>
  JSON.parse(JSON.stringify(toRaw(value)))

export const pollFormToDraft = (form: PollFormData): TopicDraftPoll => ({
  title: form.title,
  description: form.description,
  choice_type: form.choice_type,
  min_choice: Number(form.min_choice),
  max_choice: Number(form.max_choice),
  closes_at: closesAtFromPicker(form.closes_at) ?? null,
  result_visibility: form.result_visibility,
  is_anonymous: form.is_anonymous,
  can_change_vote: form.can_change_vote,
  options: form.options
    .filter((option) => option._status !== 'deleted')
    .map((option) => ({ text: option.text }))
})

export const pollFormToCreate = (form: PollFormData): PollCreate =>
  pollFormToDraft(form)

export const pollFormFromDraft = (poll: TopicDraftPoll): PollFormData => ({
  title: poll.title,
  description: poll.description,
  options: poll.options.map((option) => ({ text: option.text })),
  choice_type: poll.choice_type,
  min_choice: poll.min_choice,
  max_choice: poll.max_choice,
  closes_at: deadlineToPicker(poll.closes_at),
  result_visibility: poll.result_visibility,
  is_anonymous: poll.is_anonymous,
  can_change_vote: poll.can_change_vote
})

const positiveOrNull = (value: number) => {
  const n = Number(value)
  return n >= 1 ? n : null
}

export const lotteryFormToDraft = (
  form: LotteryFormData
): TopicDraftLottery => ({
  title: form.title,
  description: form.description,
  entry_mode: form.entry_mode,
  floor_rule: form.entry_mode === 'floor' ? form.floor_rule : null,
  draw_mode: form.draw_mode,
  draw_threshold:
    form.draw_mode === 'threshold' ? positiveOrNull(form.draw_threshold) : null,
  closes_at: closesAtFromPicker(form.deadline) ?? null,
  min_account_age_days: Number(form.min_account_age_days),
  min_moemoepoint: Number(form.min_moemoepoint),
  is_entry_list_public: form.show_entrants,
  prizes: form.prizes.map((prize) => ({
    title: prize.name,
    description: prize.description,
    image_hashes: prize.image_hashes,
    adult_image_hashes: prize.nsfw_hashes.filter((hash) =>
      prize.image_hashes.includes(hash)
    ),
    delivery: prize.delivery,
    point_mode: prize.delivery === 'point' ? prize.point_mode : null,
    point_amount:
      prize.delivery === 'point' ? positiveOrNull(prize.point_amount) : null,
    slot_count: Number(prize.slots)
  }))
})

export const lotteryFormFromDraft = (
  lottery: TopicDraftLottery
): LotteryFormData => ({
  title: lottery.title,
  description: lottery.description,
  entry_mode: lottery.entry_mode,
  floor_rule: lottery.floor_rule ?? '',
  draw_mode: lottery.draw_mode,
  draw_threshold: lottery.draw_threshold ?? 0,
  deadline: deadlineToPicker(lottery.closes_at),
  min_account_age_days: lottery.min_account_age_days,
  min_moemoepoint: lottery.min_moemoepoint,
  show_entrants: lottery.is_entry_list_public,
  prizes: lottery.prizes.map(
    (prize): LotteryPrizeFormData => ({
      name: prize.title,
      description: prize.description,
      image_hashes: [...prize.image_hashes],
      image_urls: prize.image_hashes.map(imageTokenUrl),
      nsfw_hashes: [...prize.adult_image_hashes],
      machine_nsfw_hashes: [],
      delivery: prize.delivery,
      point_mode: prize.point_mode ?? 'fixed',
      point_amount: prize.point_amount ?? 0,
      slots: prize.slot_count,
      codes: ''
    })
  )
})

export const lotteryWithoutCodes = (
  form: LotteryFormData
): LotteryFormData => ({
  ...form,
  prizes: form.prizes.map((prize) => ({ ...prize, codes: '' }))
})

export const prizeCodeCount = (prize: LotteryPrizeFormData) =>
  prize.codes.split('\n').filter((code) => code.trim()).length

export const lotteryHasCodes = (form: LotteryFormData) =>
  form.prizes.some((prize) => prizeCodeCount(prize) > 0)

export const lotteryMissesCodes = (form: LotteryFormData) =>
  form.prizes.some(
    (prize) =>
      prize.delivery === 'code' && prizeCodeCount(prize) !== Number(prize.slots)
  )

export const prizePointBudget = (prize: LotteryPrizeFormData) => {
  if (prize.delivery !== 'point') {
    return 0
  }
  const amount = Number(prize.point_amount || 0)
  return prize.point_mode === 'fixed'
    ? amount * Number(prize.slots || 0)
    : amount
}

export const lotteryPointBudget = (form: LotteryFormData) =>
  form.prizes.reduce((sum, prize) => sum + prizePointBudget(prize), 0)

const isPast = (picked?: string) => {
  const deadline = deadlineFromPicker(picked)
  return !!deadline && new Date(deadline).getTime() <= Date.now()
}

// These run before the topic is created. Anything createPoll or createLottery
// would refuse afterwards leaves a published topic without its lottery.
export const pendingPollIssue = (form: PollFormData) =>
  isPast(form.closes_at)
    ? `投票「${form.title}」的截止日期已经过去, 请修改后再发布`
    : ''

export const pendingLotteryIssue = (form: LotteryFormData) => {
  const name = `抽奖「${form.title}」`
  if (form.draw_mode === 'deadline' && !form.deadline) {
    return `${name}按时间开奖, 需要设置开奖时间`
  }
  if (isPast(form.deadline)) {
    return `${name}的开奖时间已经过去, 请修改后再发布`
  }
  if (lotteryMissesCodes(form)) {
    return `${name}的兑换码数量与名额不一致, 兑换码不会保存在草稿和浏览器里, 请重新填写`
  }
  return ''
}
