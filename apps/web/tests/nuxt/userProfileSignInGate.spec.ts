// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { computed, ref, toValue } from 'vue'
import UserPage from '~/pages/user.vue'

const { holder, startOAuthLogin, useKunSeoMeta } = vi.hoisted(() => ({
  holder: { code: null as string | null, status: 0 },
  startOAuthLogin: vi.fn(),
  useKunSeoMeta: vi.fn()
}))

mockNuxtImport('useKunApi', () => () => {
  const result = {
    data: computed(() => undefined),
    problem: computed(() => ({
      kind: 'problem',
      status: holder.status,
      code: holder.code,
      errors: [],
      requestId: null
    })),
    status: ref('error'),
    refresh: vi.fn()
  }
  return Object.assign(Promise.resolve(result), result)
})
mockNuxtImport('startOAuthLogin', () => startOAuthLogin)
mockNuxtImport('useKunSeoMeta', () => useKunSeoMeta)

const seo = () => toValue(useKunSeoMeta.mock.lastCall?.[0])

afterEach(() => {
  startOAuthLogin.mockReset()
  useKunSeoMeta.mockReset()
})

const mountRefused = (code: string, status: number) => {
  holder.code = code
  holder.status = status
  return mountSuspended(UserPage, { route: '/user/5/topic/authored' })
}

describe('user profile page for an anonymous visitor', () => {
  it('asks to sign in when the user shows their profile to signed-in users only', async () => {
    const wrapper = await mountRefused('PROFILE_SIGN_IN_REQUIRED', 403)

    expect(wrapper.text()).toContain('该用户设置了仅登录用户可查看主页')
    expect(wrapper.text()).not.toContain('未找到该用户')
    expect(seo()).toEqual({ noindex: true, title: '登录后查看该用户' })

    const login = wrapper
      .findAllComponents({ name: 'KunButton' })
      .find((button) => button.text() === '登录')
    expect(login).toBeDefined()
    await login!.trigger('click')
    expect(startOAuthLogin).toHaveBeenCalledWith({
      returnTo: '/user/5/topic/authored'
    })
  })

  it('still says the user was not found for a 404', async () => {
    const wrapper = await mountRefused('NOT_FOUND', 404)

    expect(wrapper.text()).toContain('未找到该用户')
    expect(wrapper.text()).not.toContain('仅登录用户')
    expect(seo()).toEqual({ noindex: true, title: '未找到该用户' })
  })
})
