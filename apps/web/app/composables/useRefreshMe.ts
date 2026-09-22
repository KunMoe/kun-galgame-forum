interface MeResponse {
  id: number
  sub: string
  name: string
  avatar: string
  roles: string[]
  moemoepoint: number
  bio: string
  adult_confirmed: boolean
  nsfw_display: string
}

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

    const config = useRuntimeConfig()

    inFlight = (async () => {
      const resp = await $fetch<{ code: number; data?: MeResponse }>(
        `${config.public.apiBaseUrl}/api/auth/me`,
        { credentials: 'include' }
      ).catch(() => null)
      const me = resp?.code === 0 ? resp.data : null

      if (me?.name && me.id === userStore.id) {
        userStore.setProfileInfo({
          name: me.name,
          avatar: me.avatar,
          roles: me.roles ?? [],
          adultConfirmed: me.adult_confirmed,
          nsfwDisplay: me.nsfw_display
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
