// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import type { FollowingActivityGroup } from '#shared/utils/api/schemas'
import FollowingGroup from './FollowingGroup.vue'

const { api, reportProblem } = vi.hoisted(() => ({
  api: { GET: vi.fn() },
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)

const problem = (status: number, code: string) => {
  const body = { code, status, errors: [] }
  return {
    error: body,
    response: new Response(JSON.stringify(body), {
      status,
      headers: { 'content-type': 'application/problem+json' }
    })
  }
}

const group = {
  object: 'following_activity_group',
  id: '11',
  site: 'kungal',
  actor: { object: 'user', id: '7', name: 'alice', avatar: null },
  verb: 'publish',
  object_kind: 'topic',
  object_label: '话题',
  calendar_date: '2026-09-27',
  item_count: 5,
  latest_at: '2026-09-27T08:00:00Z',
  items: []
} as unknown as FollowingActivityGroup

beforeEach(() => {
  api.GET.mockReset()
  reportProblem.mockReset()
})

const expand = async () => {
  const wrapper = await mountSuspended(FollowingGroup, {
    props: { group, includeNsfw: false }
  })
  await wrapper.findComponent({ name: 'KunButton' }).trigger('click')
  await flushPromises()
  return wrapper
}

describe('HomeFollowingGroup', () => {
  it('reports a group that is gone instead of showing an error', async () => {
    api.GET.mockResolvedValueOnce(problem(404, 'NOT_FOUND'))
    const wrapper = await expand()

    expect(wrapper.emitted('gone')).toEqual([['11']])
    expect(reportProblem).not.toHaveBeenCalled()
  })

  it('reports any other failure and keeps the group', async () => {
    api.GET.mockResolvedValueOnce(problem(503, 'SERVICE_UNAVAILABLE'))
    const wrapper = await expand()

    expect(wrapper.emitted('gone')).toBeUndefined()
    expect(reportProblem).toHaveBeenCalledOnce()
  })
})
