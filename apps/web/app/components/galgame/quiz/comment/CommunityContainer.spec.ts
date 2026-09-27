// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import CommunityContainer from './CommunityContainer.vue'

const json = (status: number, body: unknown, type = 'application/json') =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': type }
  })

const pathOf = (input: Request | string | URL) =>
  new URL(input instanceof Request ? input.url : String(input)).pathname

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
  clearNuxtData()
})

describe('GalgameQuizCommentCommunityContainer', () => {
  it('opens the discussion once the viewer answers, without a reload', async () => {
    let answered = false
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: Request | string | URL) => {
        if (!pathOf(input).endsWith('/wall-comments')) {
          return json(200, { is_following: false })
        }
        return answered
          ? json(200, { object: 'list', items: [] })
          : json(
              403,
              {
                type: 'about:blank',
                title: 'Forbidden',
                status: 403,
                code: 'QUIZ_ANSWER_REQUIRED'
              },
              'application/problem+json'
            )
      })
    )
    wrapper = await mountSuspended(CommunityContainer, {
      props: { quizId: 7001, commentCount: 0, answered: false }
    })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('作答后即可查看并参与讨论')
    })

    answered = true
    await wrapper.setProps({ answered: true })
    await vi.waitFor(() => {
      expect(wrapper!.text()).toContain('还没有人讨论这道题目')
    })
    expect(wrapper.text()).not.toContain('作答后即可查看并参与讨论')
  })
})
