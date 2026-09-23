import { settle } from '#shared/utils/api/problem'

type WorkFlags = { has_liked: boolean; has_favorited: boolean }

export const useMyGalgameInteractions = () => {
  const { id } = usePersistUserStore()
  const api = useApiClient()
  const states = useState<Record<string, WorkFlags>>(
    'my-work-states',
    () => ({})
  )
  const requested = useState<string[]>('my-work-states-requested', () => [])
  const pending = useState<string[]>('my-work-states-pending', () => [])
  const flushScheduled = useState<boolean>(
    'my-work-states-flush-scheduled',
    () => false
  )

  const fetchChunk = async (chunk: string[]) => {
    const res = await settle(
      api.GET('/me/work-states', {
        params: { query: { work_ids: chunk } }
      })
    )
    if (!res.ok) {
      const failed = new Set(chunk)
      requested.value = requested.value.filter((wid) => !failed.has(wid))
      return
    }
    const next = { ...states.value }
    for (const item of res.data.items) {
      next[item.work_id] = {
        has_liked: item.has_liked,
        has_favorited: item.has_favorited
      }
    }
    states.value = next
  }

  const flush = async () => {
    flushScheduled.value = false
    const seen = new Set(requested.value)
    const ids = [...new Set(pending.value)].filter(
      (wid) => wid.length > 0 && !seen.has(wid)
    )
    pending.value = []
    if (!id || !ids.length) return
    requested.value = [...requested.value, ...ids]
    for (let i = 0; i < ids.length; i += 100) {
      await fetchChunk(ids.slice(i, i + 100))
    }
  }

  const ensureLoaded = (workIds?: Array<string | number>) => {
    if (!id || !workIds?.length) return
    pending.value = [...pending.value, ...workIds.map(String)]
    if (flushScheduled.value) return
    flushScheduled.value = true
    queueMicrotask(() => {
      void flush()
    })
  }

  const setFavorited = (workId: string | number, isFav: boolean) => {
    const key = String(workId)
    const current = states.value[key]
    states.value = {
      ...states.value,
      [key]: {
        has_liked: current?.has_liked ?? false,
        has_favorited: isFav
      }
    }
  }

  return {
    isLiked: (workId: string | number) =>
      states.value[String(workId)]?.has_liked ?? false,
    isFavorited: (workId: string | number) =>
      states.value[String(workId)]?.has_favorited ?? false,
    setFavorited,
    ensureLoaded
  }
}
