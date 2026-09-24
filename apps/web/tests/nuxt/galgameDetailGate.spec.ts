// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { computed, nextTick, ref, toValue, type Ref } from 'vue'
import GalgameDetailPage from '~/pages/galgame/[id]/index.vue'

const holder = vi.hoisted(() => ({
  work: null as Ref<Record<string, unknown> | undefined> | null,
  key: null as unknown
}))

// setup resolves before the data lands, which is what a hydration key miss
// does: useAsyncData fetches on mount instead of awaiting.
mockNuxtImport('useApi', () => (key: unknown) => {
  holder.key = key
  holder.work ??= ref(undefined)
  const work = holder.work
  const result = {
    data: computed(() => work.value),
    problem: computed(() => null),
    status: ref('pending'),
    refresh: vi.fn()
  }
  return Object.assign(Promise.resolve(result), result)
})

const signIn = (nsfwDisplay: string) => {
  usePersistUserStore().setUserInfo({
    id: 7,
    sub: 'u-7',
    name: 'kun',
    avatar: '',
    avatarMin: '',
    moemoepoint: 0,
    roles: ['user'],
    isCheckIn: false,
    dailyToolsetUploadBytes: 0,
    adultConfirmed: true,
    nsfwDisplay
  })
}

const mountPage = () =>
  mountSuspended(GalgameDetailPage, {
    route: '/galgame/31',
    global: { stubs: { Galgame: true, KunNsfwGate: true } }
  })

afterEach(() => {
  holder.work = null
  holder.key = null
  usePersistUserStore().resetUser()
})

describe('galgame detail NSFW gate', () => {
  it('closes for a hide-stance reader when the work lands after setup', async () => {
    signIn('hide')
    const wrapper = await mountPage()
    holder.work!.value = { id: '31', is_nsfw: true, is_published: true }
    await nextTick()

    expect(wrapper.findComponent({ name: 'KunNsfwGate' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'Galgame' }).exists()).toBe(false)
  })

  it('closes when the stance narrows after the page rendered', async () => {
    signIn('show')
    const wrapper = await mountPage()
    holder.work!.value = { id: '31', is_nsfw: true, is_published: true }
    await nextTick()
    expect(wrapper.findComponent({ name: 'Galgame' }).exists()).toBe(true)

    usePersistUserStore().setContentStance(true, 'hide')
    await nextTick()
    expect(wrapper.findComponent({ name: 'KunNsfwGate' }).exists()).toBe(true)
  })

  it('keeps its data key when a signed-in stance flips', async () => {
    signIn('show')
    await mountPage()
    const before = toValue(holder.key as () => string)

    usePersistUserStore().setContentStance(true, 'hide')
    expect(toValue(holder.key as () => string)).toBe(before)
  })
})
