import { settle } from '#shared/utils/api/problem'

export type KunStanceWriteResult = 'ok' | 'unavailable' | 'failed'

export const useContentStance = () => {
  const api = useApiClient()
  const userStore = usePersistUserStore()
  const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())

  const isSignedIn = computed(() => !!userStore.id)

  // Signed in, the account decides and the cookie is ignored — the API applies
  // the same fold server-side, so a stale cookie cannot widen what comes back.
  // Signed out, the cookie is the whole preference, exactly as before.
  const stance = computed<KunContentStance>(() => {
    if (isSignedIn.value) {
      return foldContentStance(userStore.adultConfirmed, userStore.nsfwDisplay)
    }
    const limit = showKUNGalgameContentLimit.value
    return limit === 'nsfw' || limit === 'all' ? 'show' : 'hide'
  })

  const allowsNsfw = computed(() => stance.value !== 'hide')
  const isBlurred = computed(() => stance.value === 'blur')

  const setStance = async (
    next: KunContentStance
  ): Promise<KunStanceWriteResult> => {
    const result = await settle(
      api.PUT('/me/nsfw-display', {
        body: { nsfw_display: next }
      })
    )
    if (!result.ok) {
      if (result.problem.code === 'SCOPE_REQUIRED') {
        return 'unavailable'
      }
      reportProblem(result.problem)
      return 'failed'
    }

    userStore.setContentStance(true, result.data.nsfw_display)
    return 'ok'
  }

  // Signed out there is no account to write to, so the binary cookie switch
  // stays exactly what it was: one value, no round trip, no reload contract
  // change.
  const setAnonymousNsfw = (enabled: boolean) => {
    showKUNGalgameContentLimit.value = enabled ? 'nsfw' : 'sfw'
  }

  return {
    isSignedIn,
    stance,
    allowsNsfw,
    isBlurred,
    setStance,
    setAnonymousNsfw
  }
}
