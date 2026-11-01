// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises, type VueWrapper } from '@vue/test-utils'
import type {
  RatingSummary,
  WorkRecord,
  WorkRecordPut
} from '#shared/utils/api/schemas'
import RecordModal from './Modal.vue'

const { api, alert, message } = vi.hoisted(() => ({
  api: { GET: vi.fn(), PUT: vi.fn() },
  alert: vi.fn(),
  message: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('useComponentMessageStore', () => () => ({ alert }))
mockNuxtImport('useMessage', () => message)

type PutInit = { body: WorkRecordPut }

const ok = (data: unknown) => ({
  data,
  response: new Response(null, { status: 200 })
})

const rating = (): RatingSummary =>
  ({
    object: 'rating',
    id: '31',
    overall: 8,
    recommend: 'yes',
    spoiler_level: 'none',
    short_summary: '好玩',
    game_types: ['plot'],
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
  }) as unknown as RatingSummary

const record = (
  playState: WorkRecord['play_state'],
  rest: Partial<WorkRecord> = {}
): WorkRecord => ({
  object: 'work_record',
  work_id: '7',
  play_state: playState,
  minutes: 0,
  rating_summary: null,
  ...rest
})

let wrapper: VueWrapper | undefined

const mountWith = async (loaded: WorkRecord) => {
  api.GET.mockResolvedValue(ok(loaded))
  api.PUT.mockImplementation(async (_path: string, init: PutInit) =>
    ok(record(init.body.play_state ?? null))
  )
  alert.mockResolvedValue(true)
  wrapper = await mountSuspended(RecordModal, {
    props: { workId: 7, modelValue: false }
  })
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
  return wrapper
}

const openWith = async (playState: WorkRecord['play_state']) => {
  const group = (await mountWith(record(playState)))
    .findComponent({ name: 'KunRadioGroup' })
    .props() as { modelValue: string; options: { value: string }[] }
  return {
    picked: group.modelValue,
    offered: group.options.map((o) => o.value)
  }
}

// KunModal teleports its body, so the buttons are not under the mount wrapper.
const click = async (text: string) => {
  ;[...document.body.querySelectorAll('button')]
    .find((button) => (button.textContent ?? '').trim() === text)!
    .click()
  await flushPromises()
}
const save = () => click('保存')

const emitFrom = async (name: string, value: unknown) => {
  wrapper!.findComponent({ name }).vm.$emit('update:modelValue', value)
  await flushPromises()
}

const ratingProps = (name: string) =>
  wrapper!.findComponent({ name }).props() as {
    modelValue: unknown
    disabled: boolean
  }

const sent = () => (api.PUT.mock.calls.at(-1)![1] as PutInit).body
const sentPlayState = () => sent().play_state

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  api.GET.mockReset()
  api.PUT.mockReset()
  alert.mockReset()
  message.mockReset()
})

describe('a finish another app recorded without a completion', () => {
  it('is offered as 已通关 and saved back unchanged', async () => {
    const { picked, offered } = await openWith('done')

    expect(picked).toBe('done')
    expect(offered).toEqual([
      'wish',
      'doing',
      'done',
      'done_one_route',
      'done_main',
      'done_all',
      'on_hold',
      'dropped'
    ])

    await save()
    expect(sentPlayState()).toBe('done')
  })

  it('is not offered to anyone who does not have it', async () => {
    const { picked, offered } = await openWith('done_one_route')

    expect(picked).toBe('done_one_route')
    expect(offered).not.toContain('done')

    await save()
    expect(sentPlayState()).toBe('done_one_route')
  })
})

describe('the rating is the form', () => {
  it('opens unscored for a new rater, and saving unscored keeps no rating', async () => {
    await mountWith(record('doing', { minutes: 120 }))

    expect(ratingProps('KunRating').modelValue).toBe(0)
    expect(ratingProps('KunRating').disabled).toBe(false)

    await save()
    expect(sent()).toEqual({ play_state: 'doing', minutes: 120 })
    expect(alert).not.toHaveBeenCalled()
  })

  it('drops an unsaved score when reopened', async () => {
    await mountWith(record('doing'))
    await emitFrom('KunRating', 9)

    await wrapper!.setProps({ modelValue: false })
    await wrapper!.setProps({ modelValue: true })
    await flushPromises()
    expect(ratingProps('KunRating').modelValue).toBe(0)
  })

  it('asks for the overall score before saving a short review without one', async () => {
    await mountWith(record('doing'))
    await emitFrom('KunTextarea', '还没打分的短评')

    await save()
    expect(api.PUT).not.toHaveBeenCalled()
    expect(message).toHaveBeenCalledWith(
      expect.stringContaining('请先打总体评分'),
      'warn'
    )
  })

  it('saves a score as the rating', async () => {
    await mountWith(record('doing'))
    await emitFrom('KunRating', 9)
    await emitFrom('KunCheckBox', true)

    await save()
    expect(sent().rating_input).toMatchObject({ overall: 9 })
  })

  it.each([false, true])(
    'preserves inline spoilers when saving (existing: %s)',
    async (existing) => {
      const summary = '音乐很棒，||结局第一行\n第二行||，值得一玩'
      await mountWith(
        record('doing', { rating_summary: existing ? rating() : null })
      )
      if (!existing) {
        await emitFrom('KunRating', 8)
        await emitFrom('KunCheckBox', true)
      }
      await emitFrom('KunTextarea', summary)

      expect(
        document.body.querySelector('.kun-spoiler-hidden')?.textContent
      ).toBe('结局第一行\n第二行')
      await save()
      expect(sent().rating_input).toMatchObject({
        short_summary: summary,
        spoiler_level: 'none'
      })
    }
  )

  it('deletes only the rating, keeping state and playtime', async () => {
    await mountWith(
      record('done_all', { minutes: 120, rating_summary: rating() })
    )
    expect(ratingProps('KunRating').modelValue).toBe(8)
    expect(ratingProps('KunTextarea').modelValue).toBe('好玩')

    await click('删除评分')
    expect(ratingProps('KunRating').modelValue).toBe(0)

    await save()
    expect(alert).toHaveBeenCalledWith(
      '保存后会删除你的评分',
      expect.stringContaining('其它站点上也会一起消失')
    )
    expect(sent()).toEqual({ play_state: 'done_all', minutes: 120 })
  })

  it('brings a deleted rating back before saving', async () => {
    await mountWith(record('done_all', { rating_summary: rating() }))

    await click('删除评分')
    await click('恢复评分')

    await save()
    expect(alert).not.toHaveBeenCalled()
    expect(sent().rating_input).toMatchObject({
      overall: 8,
      short_summary: '好玩'
    })
  })

  it('keeps the fields in view but disabled while the state is wish', async () => {
    await mountWith(record('done_all', { rating_summary: rating() }))
    await emitFrom('KunRadioGroup', 'wish')

    expect(ratingProps('KunRating').modelValue).toBe(8)
    expect(ratingProps('KunRating').disabled).toBe(true)
    expect(ratingProps('KunTextarea').disabled).toBe(true)
    expect(document.body.textContent).toContain('想玩的作品还不能评分')

    await save()
    expect(alert).toHaveBeenCalledWith(
      '保存后会删除你的评分',
      expect.any(String)
    )
    expect(sent()).toEqual({ play_state: 'wish' })
  })
})
