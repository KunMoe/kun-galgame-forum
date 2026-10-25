// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import type { CompanyFollowState } from '#shared/utils/api/schemas'
import { useCompanyFollowState } from './useCompanyFollowState'

const { get, put, del, useMessage, reportProblem } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
  useMessage: vi.fn(),
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => ({
  GET: get,
  PUT: put,
  DELETE: del
}))
mockNuxtImport('useMessage', () => useMessage)
mockNuxtImport('reportProblem', () => reportProblem)

const followState = (isFollowing: boolean): CompanyFollowState => ({
  object: 'company_follow_state',
  id: '9',
  company_id: '9',
  is_following: isFollowing
})

const ok = (isFollowing: boolean) => ({
  data: followState(isFollowing),
  response: new Response(null, { status: 200 })
})

const cap = () => ({
  error: { status: 422, code: 'VALIDATION_FAILED', errors: [] },
  response: new Response(null, {
    status: 422,
    headers: { 'content-type': 'application/problem+json' }
  })
})

beforeEach(() => {
  clearNuxtState()
  usePersistUserStore().resetUser()
  useAuthModal().close()
  get.mockReset()
  put.mockReset()
  del.mockReset()
  useMessage.mockReset()
  reportProblem.mockReset()
})

describe('useCompanyFollowState', () => {
  it('toggle calls PUT then DELETE', async () => {
    usePersistUserStore().id = 4
    const { set, toggle } = useCompanyFollowState(9)
    set(followState(false))

    put.mockResolvedValue(ok(true))
    expect(await toggle()).toBe(1)
    expect(put).toHaveBeenCalledExactlyOnceWith(
      '/me/followed-companies/{company_id}',
      { params: { path: { company_id: '9' } } }
    )
    expect(del).not.toHaveBeenCalled()

    del.mockResolvedValue(ok(false))
    expect(await toggle()).toBe(-1)
    expect(del).toHaveBeenCalledExactlyOnceWith(
      '/me/followed-companies/{company_id}',
      { params: { path: { company_id: '9' } } }
    )
  })

  it('opens the auth modal when logged out', async () => {
    const { toggle } = useCompanyFollowState(9)
    expect(await toggle()).toBe(0)
    expect(useAuthModal().isOpen.value).toBe(true)
    expect(put).not.toHaveBeenCalled()
    expect(del).not.toHaveBeenCalled()
    expect(get).not.toHaveBeenCalled()
  })

  it('surfaces the cap message on 422', async () => {
    usePersistUserStore().id = 4
    const { set, toggle } = useCompanyFollowState(9)
    set(followState(false))
    put.mockResolvedValue(cap())

    expect(await toggle()).toBe(0)
    expect(useMessage).toHaveBeenCalledExactlyOnceWith(
      '关注的厂商已达上限（1000 家）',
      'error'
    )
    expect(reportProblem).not.toHaveBeenCalled()
  })
})
