// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'

const { refreshNuxtData, account } = vi.hoisted(() => ({
  refreshNuxtData: vi.fn(),
  account: { nsfw_display: 'hide' }
}))

mockNuxtImport('refreshNuxtData', () => refreshNuxtData)
mockNuxtImport('onNuxtReady', () => (cb: () => void) => cb())
mockNuxtImport('useApiClient', () => () => ({
  GET: async (path: string) =>
    path === '/me/account'
      ? {
          data: {
            id: '7',
            name: 'kun',
            roles: ['user'],
            avatar: null,
            content_stance: {
              is_adult_confirmed: true,
              nsfw_display: account.nsfw_display
            }
          },
          response: new Response(null, { status: 200 })
        }
      : { response: new Response(null, { status: 503 }) }
}))

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

// refreshMe skips a read within a minute of the last one.
let clock = Date.now()
const nextMinute = () => {
  clock += 120_000
  vi.setSystemTime(clock)
}

afterEach(() => {
  refreshNuxtData.mockReset()
  usePersistUserStore().resetUser()
  vi.useRealTimers()
})

describe('useRefreshMe', () => {
  it('refetches page data when the account narrowed the stance', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    nextMinute()
    signIn('show')
    account.nsfw_display = 'hide'

    await useRefreshMe().refreshMe()

    expect(usePersistUserStore().nsfwDisplay).toBe('hide')
    expect(refreshNuxtData).toHaveBeenCalledTimes(1)
  })

  it('leaves the data alone when the stance widened or held', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    nextMinute()
    signIn('hide')
    account.nsfw_display = 'show'
    await useRefreshMe().refreshMe()

    nextMinute()
    await useRefreshMe().refreshMe()

    expect(usePersistUserStore().nsfwDisplay).toBe('show')
    expect(refreshNuxtData).not.toHaveBeenCalled()
  })
})
