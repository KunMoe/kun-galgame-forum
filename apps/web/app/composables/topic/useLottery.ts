import { settle } from '#shared/utils/api/problem'
import type {
  Lottery,
  LotteryCreate,
  LotteryPatch,
  LotteryPrizeInput
} from '#shared/utils/api/schemas'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type {
  LotteryFormData,
  LotteryPrizeFormData
} from '~/components/topic/lottery/types'
import {
  closesAtFromPicker,
  deadlineToPicker
} from '~/components/topic/miniapp/deadline'

export const lotteryToForm = (lottery: Lottery): LotteryFormData => ({
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
      image_hashes: prize.images.map((image) => image.hash),
      image_urls: prize.images.map((image) => image.image?.url ?? ''),
      nsfw_hashes: prize.images
        .filter((image) => image.is_marked_adult)
        .map((image) => image.hash),
      machine_nsfw_hashes: prize.images
        .filter((image) => image.is_graded_explicit)
        .map((image) => image.hash),
      delivery: prize.delivery,
      point_mode: prize.point_mode ?? 'fixed',
      point_amount: prize.point_amount ?? 0,
      slots: prize.slot_count,
      codes: ''
    })
  )
})

const prizeToInput = (prize: LotteryPrizeFormData): LotteryPrizeInput => {
  const input: LotteryPrizeInput = {
    title: prize.name,
    description: prize.description,
    image_hashes: prize.image_hashes,
    adult_image_hashes: prize.nsfw_hashes.filter((hash) =>
      prize.image_hashes.includes(hash)
    ),
    delivery: prize.delivery,
    slot_count: Number(prize.slots)
  }
  if (prize.delivery === 'point') {
    input.point_mode = prize.point_mode
    input.point_amount = Number(prize.point_amount)
  }
  if (prize.delivery === 'code') {
    input.codes = prize.codes
      .split('\n')
      .map((code) => code.trim())
      .filter(Boolean)
  }
  return input
}

export const formToCreate = (form: LotteryFormData): LotteryCreate => {
  const body: LotteryCreate = {
    title: form.title,
    description: form.description,
    entry_mode: form.entry_mode,
    draw_mode: form.draw_mode,
    min_account_age_days: Number(form.min_account_age_days),
    min_moemoepoint: Number(form.min_moemoepoint),
    is_entry_list_public: form.show_entrants,
    prizes: form.prizes.map(prizeToInput)
  }
  if (form.entry_mode === 'floor') {
    body.floor_rule = form.floor_rule
  }
  if (form.draw_mode === 'threshold') {
    body.draw_threshold = Number(form.draw_threshold)
  }
  const closesAt = closesAtFromPicker(form.deadline)
  if (closesAt) {
    body.closes_at = closesAt
  }
  return body
}

// Only what changed is sent. The legacy edit re-sent the whole form, and the
// date-only picker turned an unchanged 10:00 deadline into 23:59:59 on every
// save.
export const formToPatch = (
  form: LotteryFormData,
  initial: Lottery,
  rewritePrizes: boolean
): LotteryPatch => {
  const before = lotteryToForm(initial)
  const patch: LotteryPatch = {}
  if (form.title !== before.title) patch.title = form.title
  if (form.description !== before.description) {
    patch.description = form.description
  }
  if (form.entry_mode !== before.entry_mode) patch.entry_mode = form.entry_mode
  if (form.entry_mode === 'floor' && form.floor_rule !== before.floor_rule) {
    patch.floor_rule = form.floor_rule
  }
  if (form.draw_mode !== before.draw_mode) patch.draw_mode = form.draw_mode
  if (
    form.draw_mode === 'threshold' &&
    Number(form.draw_threshold) !== before.draw_threshold
  ) {
    patch.draw_threshold = Number(form.draw_threshold)
  }
  if (form.deadline !== before.deadline) {
    patch.closes_at = closesAtFromPicker(form.deadline) ?? null
  }
  if (Number(form.min_account_age_days) !== before.min_account_age_days) {
    patch.min_account_age_days = Number(form.min_account_age_days)
  }
  if (Number(form.min_moemoepoint) !== before.min_moemoepoint) {
    patch.min_moemoepoint = Number(form.min_moemoepoint)
  }
  if (form.show_entrants !== before.show_entrants) {
    patch.is_entry_list_public = form.show_entrants
  }
  if (rewritePrizes) {
    patch.prizes = form.prizes.map(prizeToInput)
  }
  return patch
}

export const useLottery = (topicId: MaybeRefOrGetter<string>) => {
  const api = useApiClient()
  const createKey = useIdempotencyKey()

  const createLottery = async (body: LotteryCreate) => {
    const target = toValue(topicId)
    const result = await settle(
      api.POST('/topics/{topic_id}/lotteries', {
        params: {
          path: { topic_id: target },
          header: {
            'Idempotency-Key': createKey.take(
              `/topics/${target}/lotteries`,
              body
            )
          }
        },
        body
      })
    )
    if (result.ok) {
      createKey.clear()
    }
    return result
  }

  const updateLottery = (lotteryId: string, body: LotteryPatch) =>
    settle(
      api.PATCH('/lotteries/{lottery_id}', {
        params: { path: { lottery_id: lotteryId } },
        body
      })
    )

  const confirmThen = async <T>(
    title: string,
    detail: string,
    task: () => Promise<T>
  ) => {
    const confirmed = await useComponentMessageStore().alert(title, detail)
    return confirmed ? task() : undefined
  }

  const deleteLottery = (lotteryId: string) =>
    confirmThen(
      '确定要删除这个抽奖吗？',
      '删除后所有参与记录、中奖名单与托管的兑换码都会一并丢失, 该操作不可恢复! 未开奖的萌萌点奖池会退回给发起人。',
      () =>
        settle(
          api.DELETE('/lotteries/{lottery_id}', {
            params: { path: { lottery_id: lotteryId } }
          })
        )
    )

  const drawNow = (lotteryId: string) =>
    confirmThen(
      '确定现在开奖吗？',
      '开奖后中奖名单立即固定, 无法撤销, 也不能再修改奖项。',
      () => updateLottery(lotteryId, { state: 'drawn' })
    )

  const cancel = (lotteryId: string) =>
    confirmThen(
      '确定要取消这个抽奖吗？',
      '取消后将不再开奖, 已参与的用户不会获得任何奖品。萌萌点奖池会全额退回给发起人。',
      () => updateLottery(lotteryId, { state: 'cancelled' })
    )

  const enter = (lotteryId: string) =>
    settle(
      api.PUT('/lotteries/{lottery_id}/entries/me', {
        params: { path: { lottery_id: lotteryId } }
      })
    )

  const withdraw = (lotteryId: string) =>
    settle(
      api.DELETE('/lotteries/{lottery_id}/entries/me', {
        params: { path: { lottery_id: lotteryId } }
      })
    )

  const listEntries = (lotteryId: string, cursor?: string) =>
    settle(
      api.GET('/lotteries/{lottery_id}/entries', {
        params: {
          path: { lottery_id: lotteryId },
          query: { limit: 100, ...(cursor ? { cursor } : {}) }
        }
      })
    )

  // POST on purpose. The code must not travel on a page-load fetch: Nuxt inlines
  // every SSR payload into the __NUXT__ blob, which is readable in page source.
  const revealCode = (lotteryId: string) =>
    settle(
      api.POST('/lotteries/{lottery_id}/code-reveals', {
        params: { path: { lottery_id: lotteryId } }
      })
    )

  const setFulfillment = (
    lotteryId: string,
    winnerId: string,
    fulfillment: 'pending' | 'shipped' | 'received' | 'forfeited'
  ) =>
    settle(
      api.PATCH('/lotteries/{lottery_id}/winners/{winner_id}', {
        params: { path: { lottery_id: lotteryId, winner_id: winnerId } },
        body: { fulfillment }
      })
    )

  return {
    createLottery,
    updateLottery,
    deleteLottery,
    drawNow,
    cancel,
    enter,
    withdraw,
    listEntries,
    revealCode,
    setFulfillment
  }
}
