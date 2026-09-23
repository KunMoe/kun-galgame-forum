import { settle } from '#shared/utils/api/problem'

interface CachedFlag {
  has: boolean
  until: number
  include_nsfw: boolean
}

const STORAGE_KEY = 'kun-galgame-release-today'

const requested = new Set<string>()

const read = (includeNsfw: boolean): CachedFlag | null => {
  if (!import.meta.client) {
    return null
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const cached = raw ? (JSON.parse(raw) as CachedFlag) : null
    return cached &&
      cached.include_nsfw === includeNsfw &&
      cached.until > Date.now()
      ? cached
      : null
  } catch {
    return null
  }
}

const write = (cached: CachedFlag) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cached))
  } catch {
    // Storage disabled (private mode). Asking again next load is the old
    // behaviour, not a failure.
  }
}

export const useGalgameReleaseToday = () => {
  const { allowsNsfw } = useContentStance()
  const hasReleaseToday = useState('galgame-release-today', () => false)
  const api = useApiClient()

  onMounted(async () => {
    // Keyed on include_nsfw, not the cookie: the API filters this flag by the
    // account now, so a cache keyed on the cookie would answer for the wrong
    // one the moment the two disagree.
    const includeNsfw = allowsNsfw.value
    const cached = read(includeNsfw)
    if (cached) {
      hasReleaseToday.value = cached.has
      return
    }
    const key = includeNsfw ? '1' : '0'
    if (requested.has(key)) {
      return
    }
    requested.add(key)

    const result = await settle(
      api.GET('/release-calendar/today', {
        params: { query: { include_nsfw: includeNsfw } }
      })
    )
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    hasReleaseToday.value = result.data.has_release
    write({
      has: result.data.has_release,
      until: Date.now() + result.data.expires_in * 1000,
      include_nsfw: includeNsfw
    })
  })

  return { hasReleaseToday }
}
