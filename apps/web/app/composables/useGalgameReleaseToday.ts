interface ReleaseTodayFlag {
  today: string
  has_release: boolean
  expires_in: number
}

interface CachedFlag {
  has: boolean
  until: number
  limit: string
}

const STORAGE_KEY = 'kun-galgame-release-today'

let requested = false

const read = (limit: string): CachedFlag | null => {
  if (!import.meta.client) {
    return null
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const cached = raw ? (JSON.parse(raw) as CachedFlag) : null
    return cached && cached.limit === limit && cached.until > Date.now()
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
  const { stance } = useContentStance()
  const hasReleaseToday = useState('galgame-release-today', () => false)

  onMounted(async () => {
    // Keyed on the stance, not the cookie: the API filters this flag by the
    // account now, so a cache keyed on the cookie would answer for the wrong
    // one the moment the two disagree.
    const limit = stance.value
    const cached = read(limit)
    if (cached) {
      hasReleaseToday.value = cached.has
      return
    }
    if (requested) {
      return
    }
    requested = true

    const flag = await kunFetch<ReleaseTodayFlag>('/galgame/calendar/today')
    if (!flag) {
      return
    }
    hasReleaseToday.value = flag.has_release
    write({
      has: flag.has_release,
      until: Date.now() + flag.expires_in * 1000,
      limit
    })
  })

  return { hasReleaseToday }
}
