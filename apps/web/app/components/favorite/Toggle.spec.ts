// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import FavoriteToggle from './Toggle.vue'

const { kunFetch } = vi.hoisted(() => ({
  kunFetch: vi.fn()
}))

mockNuxtImport('kunFetch', () => kunFetch)

afterEach(() => {
  kunFetch.mockReset()
  usePersistUserStore().resetUser()
})

describe('FavoriteToggle', () => {
  it('calls kunFetch with the endpoint and body when action is absent', async () => {
    kunFetch.mockResolvedValue('ok')
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(FavoriteToggle, {
      props: {
        favorited: false,
        count: 3,
        endpoint: '/example/favorite',
        body: { id: 9 }
      }
    })
    const reaction = wrapper.findComponent({ name: 'KunReaction' })
    await reaction.vm.$emit('change', true)
    expect(kunFetch).toHaveBeenCalledWith('/example/favorite', {
      method: 'PUT',
      body: { id: 9 }
    })
  })

  it('calls action instead of kunFetch when action is present', async () => {
    const action = vi.fn(async () => true)
    usePersistUserStore().id = 1
    const wrapper = await mountSuspended(FavoriteToggle, {
      props: {
        favorited: true,
        count: 4,
        action
      }
    })
    const reaction = wrapper.findComponent({ name: 'KunReaction' })
    await reaction.vm.$emit('change', false)
    expect(action).toHaveBeenCalledWith(false)
    expect(kunFetch).not.toHaveBeenCalled()
  })
})
