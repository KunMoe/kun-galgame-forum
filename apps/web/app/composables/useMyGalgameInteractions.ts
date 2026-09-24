import { settle } from '#shared/utils/api/problem'
import type { MyWork, WorkViewerPlaytime } from '#shared/utils/api/schemas'

// A hidden tab has nobody looking at its hearts: a browser restoring a batch of
// detail tabs spent a reader's whole catalog quota on 2026-09-24.
const whenVisible = (): Promise<void> => {
  if (document.visibilityState !== 'hidden') return Promise.resolve()
  return new Promise((resolve) => {
    const onChange = () => {
      if (document.visibilityState === 'hidden') return
      document.removeEventListener('visibilitychange', onChange)
      resolve()
    }
    document.addEventListener('visibilitychange', onChange)
  })
}

export const useMyGalgameInteractions = () => {
  const { id } = usePersistUserStore()
  const api = useApiClient()
  const states = useState<Record<string, MyWork>>('my-works', () => ({}))
  const favoritedAfterSave = useState<Record<string, boolean>>(
    'my-works-favorited-after-save',
    () => ({})
  )
  const requested = useState<string[]>('my-works-requested', () => [])
  const pending = useState<string[]>('my-works-pending', () => [])
  const flushScheduled = useState<boolean>(
    'my-works-flush-scheduled',
    () => false
  )

  const fetchChunk = async (chunk: string[]) => {
    const res = await settle(
      api.GET('/me/works', {
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
      next[item.work_id] = item
    }
    states.value = next
    const answered = new Set(res.data.items.map((item) => item.work_id))
    favoritedAfterSave.value = Object.fromEntries(
      Object.entries(favoritedAfterSave.value).filter(
        ([wid]) => !answered.has(wid)
      )
    )
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
    await whenVisible()
    for (let i = 0; i < ids.length; i += 100) {
      await fetchChunk(ids.slice(i, i + 100))
    }
  }

  const ensureLoaded = (workIds?: Array<string | number>) => {
    if (!import.meta.client || !id || !workIds?.length) return
    pending.value = [...pending.value, ...workIds.map(String)]
    if (flushScheduled.value) return
    flushScheduled.value = true
    queueMicrotask(() => {
      void flush()
    })
  }

  // After a folder write the answer is re-read from catalog; until it lands the
  // saved value stands in for it.
  const setFavorited = (workId: string | number, isFav: boolean) => {
    const key = String(workId)
    favoritedAfterSave.value = { ...favoritedAfterSave.value, [key]: isFav }
    requested.value = requested.value.filter((wid) => wid !== key)
    ensureLoaded([key])
  }

  return {
    isLiked: (workId: string | number) =>
      states.value[String(workId)]?.has_liked ?? false,
    // null is unknown — still loading, or catalog could not be read — and is
    // never drawn as "not collected".
    isFavorited: (workId: string | number): boolean | null => {
      if (!id) return false
      const key = String(workId)
      const saved = favoritedAfterSave.value[key]
      if (saved !== undefined) return saved
      const library = states.value[key]?.library
      return library ? library.collection_ids.length > 0 : null
    },
    // undefined is unknown; null is "no playtime and no play state".
    playtimeOf: (
      workId: string | number
    ): WorkViewerPlaytime | null | undefined => {
      const library = states.value[String(workId)]?.library
      return library ? library.playtime : undefined
    },
    setFavorited,
    ensureLoaded
  }
}
