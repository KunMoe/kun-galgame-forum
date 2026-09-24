// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import FavoriteToggle from './Toggle.vue'

const { message, openAuth } = vi.hoisted(() => ({
  message: vi.fn(),
  openAuth: vi.fn()
}))

mockNuxtImport('useMessage', () => message)
mockNuxtImport('useAuthModal', () => () => ({ open: openAuth }))

afterEach(() => {
  message.mockReset()
  openAuth.mockReset()
  usePersistUserStore().resetUser()
})

const mountToggle = (action: (next: boolean) => Promise<boolean>) =>
  mountSuspended(FavoriteToggle, {
    props: {
      favorited: false,
      count: 3,
      action,
      messages: ['收藏成功', '取消收藏成功'] as [string, string]
    }
  })

const toggleOn = async (wrapper: Awaited<ReturnType<typeof mountToggle>>) => {
  const reaction = wrapper.findComponent({ name: 'KunReaction' })
  reaction.vm.$emit('update:modelValue', true)
  reaction.vm.$emit('update:count', 4)
  reaction.vm.$emit('change', true)
  await new Promise((resolve) => setTimeout(resolve, 0))
  return reaction
}

describe('FavoriteToggle', () => {
  it('calls action, then reports success and emits changed', async () => {
    usePersistUserStore().id = 1
    const action = vi.fn(async () => true)
    const wrapper = await mountToggle(action)

    const reaction = await toggleOn(wrapper)

    expect(action).toHaveBeenCalledWith(true)
    expect(message).toHaveBeenCalledWith('收藏成功', 'success')
    expect(wrapper.emitted('changed')).toEqual([[true]])
    expect(reaction.props('modelValue')).toBe(true)
    expect(reaction.props('count')).toBe(4)
  })

  it('reverts the optimistic toggle when action fails', async () => {
    usePersistUserStore().id = 1
    const action = vi.fn(async () => false)
    const wrapper = await mountToggle(action)

    const reaction = await toggleOn(wrapper)

    expect(action).toHaveBeenCalledWith(true)
    expect(wrapper.emitted('changed')).toBeUndefined()
    expect(message).not.toHaveBeenCalled()
    expect(reaction.props('modelValue')).toBe(false)
    expect(reaction.props('count')).toBe(3)
  })

  it('asks a signed-out user to sign in instead of calling action', async () => {
    const action = vi.fn(async () => true)
    const wrapper = await mountToggle(action)

    const reaction = await toggleOn(wrapper)

    expect(openAuth).toHaveBeenCalled()
    expect(action).not.toHaveBeenCalled()
    expect(reaction.props('modelValue')).toBe(false)
    expect(reaction.props('count')).toBe(3)
  })
})
