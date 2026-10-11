// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mockNuxtImport, mountSuspended } from '@nuxt/test-utils/runtime'
import Panel from './Panel.vue'

const { api, startOAuthLogin } = vi.hoisted(() => ({
  api: { GET: vi.fn() },
  startOAuthLogin: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('startOAuthLogin', () => startOAuthLogin)

const refused = (code: string, status: number) => {
  const body = { code, status, errors: [] }
  return {
    error: body,
    response: new Response(JSON.stringify(body), {
      status,
      headers: { 'content-type': 'application/problem+json' }
    })
  }
}

afterEach(() => {
  api.GET.mockReset()
  startOAuthLogin.mockReset()
  useState<Record<string, unknown>>('user-card').value = {}
})

const mountPanel = async (userId: number) => {
  const wrapper = await mountSuspended(Panel, { props: { userId } })
  await flushPromises()
  return wrapper
}

describe('UserHoverCardPanel', () => {
  it('asks an anonymous reader to sign in instead of reporting a failure', async () => {
    api.GET.mockResolvedValueOnce(refused('PROFILE_SIGN_IN_REQUIRED', 403))
    const wrapper = await mountPanel(5)

    expect(wrapper.text()).toContain('该用户设置了仅登录用户可查看主页')
    expect(wrapper.text()).not.toContain('用户信息加载失败')

    await wrapper.findComponent({ name: 'KunButton' }).trigger('click')
    expect(startOAuthLogin).toHaveBeenCalledOnce()
  })

  it('remembers the refusal like a missing user, without asking again', async () => {
    api.GET.mockResolvedValueOnce(refused('PROFILE_SIGN_IN_REQUIRED', 403))
    await mountPanel(6)
    await mountPanel(6)

    expect(api.GET).toHaveBeenCalledOnce()
  })
})
