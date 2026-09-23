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

    inFlight = (async () => {
      const result = await settle(api.GET('/me/account'))
      const me = result.ok ? result.data : null

      if (me?.name && Number(me.id) === userStore.id) {
        userStore.setProfileInfo({
          name: me.name,
          avatar: me.avatar?.url ?? '',
          roles: me.roles,
          adultConfirmed: me.content_stance?.is_adult_confirmed,
          nsfwDisplay: me.content_stance?.nsfw_display
        })
      }
      lastFetchedAt = Date.now()
    })().finally(() => {
      inFlight = null
    })

    return inFlight
  }

  return { refreshMe }
}
