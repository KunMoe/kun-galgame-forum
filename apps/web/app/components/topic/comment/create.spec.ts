// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import TopicCommentPanel from './Panel.vue'

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
  params: { path: { reply_id: string }; header: Record<string, string> }
  body: { text: string; parent_comment_id?: string }
}

const calls: { path: string; init: PostInit }[] = []

const unavailable = () => {
  const body = { code: 'SERVICE_UNAVAILABLE', status: 503, errors: [] }
  return {
    error: body,
    response: new Response(JSON.stringify(body), {
      status: 503,
      headers: { 'content-type': 'application/problem+json' }
    })
  }
}

api.POST.mockImplementation(async (path: string, init: PostInit) => {
  calls.push({ path, init })
  return unavailable()
})

const publish = async (
  wrapper: Awaited<ReturnType<typeof mountSuspended>>,
  text: string
) => {
  await wrapper.find('textarea').setValue(text)
  const button = wrapper
    .findAll('button')
    .find((candidate: { text: () => string }) =>
      candidate.text().includes('发布评论')
    )!
  await button.trigger('click')
}

afterEach(() => {
  calls.length = 0
  usePersistUserStore().resetUser()
})

// K12: the server fingerprints method + path + body. A key the client kept
// after a failure and then sent to another reply answers
// 409 IDEMPOTENCY_KEY_REUSED for the next 24 hours.
describe('creating a comment folds the target into the idempotency key', () => {
  it('keeps the key when retrying the same reply and renews it for another', async () => {
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(TopicCommentPanel, {
      props: { replyId: '11', targetUser: { id: 2, name: 'u2', avatar: '' } }
    })

    await publish(wrapper, 'hello')
    await vi.waitFor(() => expect(calls).toHaveLength(1))
    await publish(wrapper, 'hello')
    await vi.waitFor(() => expect(calls).toHaveLength(2))

    await wrapper.setProps({ replyId: '12' })
    await publish(wrapper, 'hello')
    await vi.waitFor(() => expect(calls).toHaveLength(3))

    const keyOf = (index: number) =>
      calls[index]!.init.params.header['Idempotency-Key']
    expect(calls[0]!.path).toBe('/replies/{reply_id}/comments')
    expect(calls[0]!.init.params.path.reply_id).toBe('11')
    expect(calls[2]!.init.params.path.reply_id).toBe('12')
    expect(keyOf(0)).toBeTruthy()
    expect(keyOf(1)).toBe(keyOf(0))
    expect(keyOf(2)).not.toBe(keyOf(0))
    wrapper.unmount()
  })

  it('sends the parent comment id and no topic or target user', async () => {
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(TopicCommentPanel, {
      props: {
        replyId: '11',
        parentCommentId: '5',
        targetUser: { id: 2, name: 'u2', avatar: '' }
      }
    })
    await publish(wrapper, 'hello')
    await vi.waitFor(() => expect(calls).toHaveLength(1))
    expect(calls[0]!.init.body).toEqual({
      text: 'hello',
      parent_comment_id: '5'
    })
    wrapper.unmount()
  })

  it('refuses an empty body before it reaches the network', async () => {
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(TopicCommentPanel, {
      props: { replyId: '11', targetUser: { id: 2, name: 'u2', avatar: '' } }
    })
    await publish(wrapper, '   ')
    expect(calls).toHaveLength(0)
    wrapper.unmount()
  })
})
