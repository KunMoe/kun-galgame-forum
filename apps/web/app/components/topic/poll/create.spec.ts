// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { DOMWrapper, type VueWrapper } from '@vue/test-utils'
import TopicPollModal from './Modal.vue'

// happy-dom's Request constructor drops a Headers instance passed in its init,
// so a stubbed global fetch never sees Idempotency-Key. The header is asserted
// where openapi-fetch receives it instead.
const { api, reportProblem } = vi.hoisted(() => ({
  api: { POST: vi.fn() },
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)

// A signed-in store starts the cloud-preferences sync, which calls api.GET on
// this POST-only stub.
mockNuxtImport('useCloudPreferences', () => () => ({
  sync: async () => {},
  flush: async () => {}
}))

type PostInit = {
  params: { path: { topic_id: string }; header: Record<string, string> }
  body: { title: string; options: { text: string }[] }
}

const calls: { path: string; init: PostInit }[] = []

api.POST.mockImplementation(async (path: string, init: PostInit) => {
  calls.push({ path, init })
  const body = { code: 'SERVICE_UNAVAILABLE', status: 503, errors: [] }
  return {
    error: body,
    response: new Response(JSON.stringify(body), {
      status: 503,
      headers: { 'content-type': 'application/problem+json' }
    })
  }
})

let wrapper: VueWrapper | undefined

const buttonSaying = (text: string) =>
  [...document.body.querySelectorAll('button')].find(
    (button) => (button.textContent ?? '').trim() === text
  )

// KunModal teleports its body, so the form is not under the mount wrapper.
const publish = async () => {
  const inputs = [...document.body.querySelectorAll('input')].map(
    (element) => new DOMWrapper(element)
  )
  await inputs[0]!.setValue('今晚吃什么')
  await inputs[1]!.setValue('拉面')
  await inputs[2]!.setValue('咖喱')
  buttonSaying('发布投票')!.click()
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  calls.length = 0
  reportProblem.mockClear()
  usePersistUserStore().resetUser()
})

// K12: the server fingerprints method + path + body. A key kept after a failure
// and then sent to another topic answers 409 IDEMPOTENCY_KEY_REUSED for the
// next 24 hours.
describe('creating a poll folds the topic into the idempotency key', () => {
  it('keeps the key for a retry and renews it for another topic', async () => {
    usePersistUserStore().id = 1
    wrapper = await mountSuspended(TopicPollModal, {
      props: { topicId: '42', modelValue: true }
    })

    await publish()
    await vi.waitFor(() => expect(calls).toHaveLength(1))
    await publish()
    await vi.waitFor(() => expect(calls).toHaveLength(2))

    await wrapper.setProps({ topicId: '43' })
    await publish()
    await vi.waitFor(() => expect(calls).toHaveLength(3))

    const keyOf = (index: number) =>
      calls[index]!.init.params.header['Idempotency-Key']
    expect(calls[0]!.path).toBe('/topics/{topic_id}/polls')
    expect(calls[0]!.init.params.path.topic_id).toBe('42')
    expect(calls[2]!.init.params.path.topic_id).toBe('43')
    expect(keyOf(0)).toBeTruthy()
    expect(keyOf(1)).toBe(keyOf(0))
    expect(keyOf(2)).not.toBe(keyOf(0))
  })

  it('sends choice_type and options, never type, deadline or status', async () => {
    usePersistUserStore().id = 1
    wrapper = await mountSuspended(TopicPollModal, {
      props: { topicId: '42', modelValue: true }
    })
    await publish()
    await vi.waitFor(() => expect(calls).toHaveLength(1))

    const body = calls[0]!.init.body as unknown as Record<string, unknown>
    expect(body.choice_type).toBe('single')
    expect(body).not.toHaveProperty('type')
    expect(body).not.toHaveProperty('deadline')
    expect(body).not.toHaveProperty('status')
    expect(body).not.toHaveProperty('topic_id')
    expect(body.options).toEqual([{ text: '拉面' }, { text: '咖喱' }])
  })
})
