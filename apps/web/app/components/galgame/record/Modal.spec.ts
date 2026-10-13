// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises, type VueWrapper } from '@vue/test-utils'
import type { WorkRecord, WorkRecordPut } from '#shared/utils/api/schemas'
import RecordModal from './Modal.vue'

const { api } = vi.hoisted(() => ({
  api: { GET: vi.fn(), PUT: vi.fn() }
}))

mockNuxtImport('useApiClient', () => () => api)

type PutInit = { body: WorkRecordPut }

const ok = (data: unknown) => ({
  data,
  response: new Response(null, { status: 200 })
})

const record = (playState: WorkRecord['play_state']): WorkRecord => ({
  object: 'work_record',
  work_id: '7',
  play_state: playState,
  minutes: 0,
  rating_summary: null
})

let wrapper: VueWrapper | undefined

const openWith = async (playState: WorkRecord['play_state']) => {
  api.GET.mockResolvedValue(ok(record(playState)))
  api.PUT.mockImplementation(async (_path: string, init: PutInit) =>
    ok(record(init.body.play_state ?? null))
  )
  wrapper = await mountSuspended(RecordModal, {
    props: { workId: 7, modelValue: false }
  })
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
  const group = wrapper.findComponent({ name: 'KunRadioGroup' }).props() as {
    modelValue: string
    options: { value: string }[]
  }
  return {
    picked: group.modelValue,
    offered: group.options.map((o) => o.value)
  }
}

// KunModal teleports its body, so the buttons are not under the mount wrapper.
const save = async () => {
  ;[...document.body.querySelectorAll('button')]
    .find((button) => (button.textContent ?? '').trim() === '保存')!
    .click()
  await flushPromises()
}

const sentPlayState = () =>
  (api.PUT.mock.calls.at(-1)![1] as PutInit).body.play_state

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  api.GET.mockReset()
  api.PUT.mockReset()
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
