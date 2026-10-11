// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import ProfileVisibility from './ProfileVisibility.vue'

const { api, reportProblem } = vi.hoisted(() => ({
  api: { GET: vi.fn(), PUT: vi.fn() },
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)

const settings = (isSignInRequired: boolean) => {
  const body = {
    object: 'profile_settings',
    is_sign_in_required: isSignInRequired
  }
  return {
    data: body,
    response: new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'content-type': 'application/json' }
    })
  }
}

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

beforeEach(() => {
  api.GET.mockReset()
  api.PUT.mockReset()
  reportProblem.mockReset()
})

const mountSwitch = async () => {
  const wrapper = await mountSuspended(ProfileVisibility)
  await flushPromises()
  return wrapper.findComponent({ name: 'KunSwitch' })
}

describe('UserSettingProfileVisibility', () => {
  it('shows the stored switch and saves a toggle', async () => {
    api.GET.mockResolvedValueOnce(settings(false))
    api.PUT.mockResolvedValueOnce(settings(true))
    const toggle = await mountSwitch()

    expect(api.GET).toHaveBeenCalledWith('/me/profile-settings')
    expect(toggle.props('modelValue')).toBe(false)
    expect(toggle.props('disabled')).toBe(false)

    toggle.vm.$emit('update:modelValue', true)
    await flushPromises()

    expect(api.PUT).toHaveBeenCalledWith('/me/profile-settings', {
      body: { is_sign_in_required: true }
    })
    expect(toggle.props('modelValue')).toBe(true)
    expect(reportProblem).not.toHaveBeenCalled()
  })

  it('rolls the switch back when saving fails', async () => {
    api.GET.mockResolvedValueOnce(settings(true))
    api.PUT.mockResolvedValueOnce(unavailable())
    const toggle = await mountSwitch()

    toggle.vm.$emit('update:modelValue', false)
    await flushPromises()

    expect(toggle.props('modelValue')).toBe(true)
    expect(reportProblem).toHaveBeenCalledOnce()
    expect(toggle.props('disabled')).toBe(false)
  })

  it('keeps the switch disabled when the stored value cannot be read', async () => {
    api.GET.mockResolvedValueOnce(unavailable())
    const toggle = await mountSwitch()

    expect(toggle.props('disabled')).toBe(true)
    expect(reportProblem).toHaveBeenCalledOnce()
  })
})
