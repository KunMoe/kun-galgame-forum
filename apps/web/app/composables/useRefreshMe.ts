import { settle } from '#shared/utils/api/problem'

const STALE_MS = 60_000
let lastFetchedAt = 0
let inFlight: Promise<void> | null = null

export const useRefreshMe = () => {
  const refreshMe = (): Promise<void> => {
    if (!import.meta.client) return Promise.resolve()

    const userStore = usePersistUserStore()
    if (!userStore.id) return Promise.resolve()
    if (inFlight) return inFlight
    if (Date.now() - lastFetchedAt < STALE_MS) return Promise.resolve()

    const api = useApiClient()
    const nuxtApp = useNuxtApp()
    const allowsNsfw = () =>
      foldContentStance(userStore.adultConfirmed, userStore.nsfwDisplay) !==
      'hide'

    inFlight = (async () => {
      const result = await settle(api.GET('/me/account'))
      const me = result.ok ? result.data : null

      if (me?.name && Number(me.id) === userStore.id) {
        const allowedBefore = allowsNsfw()
        userStore.setProfileInfo({
          name: me.name,
          avatar: me.avatar?.url ?? '',
          roles: me.roles,
          adultConfirmed: me.content_stance?.is_adult_confirmed,
          nsfwDisplay: me.content_stance?.nsfw_display
        })
        // Stance keys no longer move when the account's stance does, so data
        // fetched under a wider stance than the account now allows would stay
        // on screen until the next navigation.
        if (allowedBefore && !allowsNsfw()) {
          nuxtApp.runWithContext(() =>
            onNuxtReady(() => nuxtApp.runWithContext(() => refreshNuxtData()))
          )
        }
      }
      lastFetchedAt = Date.now()
    })().finally(() => {
      inFlight = null
    })

    return inFlight
  }

  return { refreshMe }
}
