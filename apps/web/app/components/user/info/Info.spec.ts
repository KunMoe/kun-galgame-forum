// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { UserProfile } from '#shared/utils/api/schemas'
import Info from './Info.vue'

const user = (id: string) =>
  ({
    id,
    created_at: '2023-10-29T07:10:00Z',
    bio: '',
    counts: {}
  }) as unknown as UserProfile

const serveMinutes = (minutes: number | null) =>
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({ object: 'user_playtime_total', minutes }),
          { status: 200, headers: { 'content-type': 'application/json' } }
        )
    )
  )

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('UserInfo', () => {
  it('leads with the total playtime when catalog reports one', async () => {
    serveMinutes(7530)
    const wrapper = await mountSuspended(Info, { props: { user: user('91') } })
    await flushPromises()
    const text = wrapper.text()
    expect(text).toContain('总游玩时长')
    expect(text).toContain('125')
    expect(text).toContain('125 小时 30 分钟')
  })

  it('leaves the card out while catalog reports no total', async () => {
    serveMinutes(null)
    const wrapper = await mountSuspended(Info, { props: { user: user('92') } })
    await flushPromises()
    expect(wrapper.text()).not.toContain('总游玩时长')
  })
})
