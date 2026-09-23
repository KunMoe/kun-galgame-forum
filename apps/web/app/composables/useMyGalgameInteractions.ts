export const useMyGalgameInteractions = () => {
  const { id } = usePersistUserStore()
  const liked = useState<number[]>('my-galgame-liked', () => [])
  const favorited = useState<number[]>('my-galgame-favorited', () => [])
  const likedLoaded = useState<boolean>(
    'my-galgame-liked-loaded',
    () => false
  )
  const pending = useState<number[]>('my-galgame-fav-pending', () => [])
  const flushScheduled = useState<boolean>(
    'my-galgame-fav-flush-scheduled',
    () => false
  )

  const mergeFavorited = (ids: number[]) => {
    if (!ids.length) return
    const set = new Set(favorited.value)
    for (const workId of ids) {
      set.add(workId)
    }
    favorited.value = [...set]
  }

  const flushFavorited = async () => {
    flushScheduled.value = false
    const ids = [...new Set(pending.value)].filter((workId) => workId > 0)
    pending.value = []
    if (!id) return
    if (!ids.length && likedLoaded.value) return

    const chunks: number[][] = []
    for (let i = 0; i < ids.length; i += 100) {
      chunks.push(ids.slice(i, i + 100))
    }
    if (!chunks.length) {
      chunks.push([])
    }

    for (const chunk of chunks) {
      const query = chunk.length ? { work_ids: chunk.join(',') } : undefined
      const res = await kunFetch<{ liked: number[]; favorited: number[] }>(
        '/galgame/interactions/mine',
        query ? { query } : undefined
      )
      if (!res) {
        pending.value = [...new Set([...pending.value, ...ids])]
        return
      }
      likedLoaded.value = true
      liked.value = res.liked ?? []
      mergeFavorited(res.favorited ?? [])
    }
  }

  const ensureLoaded = async (workIds?: number[]) => {
    if (!id) return
    if (workIds?.length) {
      pending.value = [...pending.value, ...workIds]
    }
    if (likedLoaded.value && !pending.value.length) return
    if (flushScheduled.value) return
    flushScheduled.value = true
    queueMicrotask(() => {
      void flushFavorited()
    })
  }

  const likedSet = computed(() => new Set(liked.value))
  const favoritedSet = computed(() => new Set(favorited.value))

  const setFavorited = (workId: number, isFav: boolean) => {
    const set = new Set(favorited.value)
    if (isFav) {
      set.add(workId)
    } else {
      set.delete(workId)
    }
    favorited.value = [...set]
  }

  return {
    isLiked: (workId: number) => likedSet.value.has(workId),
    isFavorited: (workId: number) => favoritedSet.value.has(workId),
    setFavorited,
    ensureLoaded
  }
}
