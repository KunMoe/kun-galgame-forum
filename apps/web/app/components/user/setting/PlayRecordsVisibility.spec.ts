// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import PlayRecordsVisibility from './PlayRecordsVisibility.vue'

const { api, reportProblem } = vi.hoisted(() => ({
  api: { GET: vi.fn(), PATCH: vi.fn() },
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)

const settings = (isPublic: boolean) => {
  const body = {
    object: 'privacy_settings',
    is_play_records_public: isPublic,
    changed_at: null
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
  api.PATCH.mockReset()
  reportProblem.mockReset()
})

const mountSwitch = async () => {
  const wrapper = await mountSuspended(PlayRecordsVisibility)
  await flushPromises()
  return wrapper.findComponent({ name: 'KunSwitch' })
}

describe('UserSettingPlayRecordsVisibility', () => {
  it('shows the stored switch and patches a toggle', async () => {
    api.GET.mockResolvedValueOnce(settings(true))
    api.PATCH.mockResolvedValueOnce(settings(false))
    const toggle = await mountSwitch()

    expect(api.GET).toHaveBeenCalledWith('/me/privacy-settings')
    expect(toggle.props('modelValue')).toBe(true)

    toggle.vm.$emit('update:modelValue', false)
    await flushPromises()

    expect(api.PATCH).toHaveBeenCalledWith('/me/privacy-settings', {
      body: { is_play_records_public: false }
    })
    expect(toggle.props('modelValue')).toBe(false)
    expect(reportProblem).not.toHaveBeenCalled()
  })

  it('rolls the switch back when saving fails', async () => {
    api.GET.mockResolvedValueOnce(settings(true))
    api.PATCH.mockResolvedValueOnce(unavailable())
    const toggle = await mountSwitch()

    toggle.vm.$emit('update:modelValue', false)
    await flushPromises()

    expect(toggle.props('modelValue')).toBe(true)
    expect(reportProblem).toHaveBeenCalledOnce()
  })
})
