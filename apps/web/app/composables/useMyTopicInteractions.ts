import { settle } from '#shared/utils/api/problem'

interface MyTopicState {
  favorited: boolean
  reactions: string[]
}

export const useMyTopicInteractions = () => {
  const { id } = usePersistUserStore()
  const api = useApiClient()
  const states = useState<Record<number, MyTopicState>>(
    'my-topic-states',
    () => ({})
  )
  const requested = useState<number[]>('my-topic-states-requested', () => [])
  const pending = useState<number[]>('my-topic-states-pending', () => [])
  const flushScheduled = useState<boolean>(
    'my-topic-states-flush-scheduled',
    () => false
  )

  const fetchChunk = async (chunk: number[]) => {
    const res = await settle(
      api.GET('/me/topic-states', {
        params: { query: { topic_ids: chunk.map(String) } }
      })
    )
    if (!res.ok) {
      const failed = new Set(chunk)
      requested.value = requested.value.filter((tid) => !failed.has(tid))
      return
    }
    const next = { ...states.value }
    for (const item of res.data.items) {
      next[Number(item.topic_id)] = {
        favorited: item.has_favorited,
        reactions: [...item.reaction_tokens]
      }
    }
    states.value = next
  }

  const flush = async () => {
    flushScheduled.value = false
    const seen = new Set(requested.value)
    const ids = [...new Set(pending.value)].filter(
      (tid) => tid > 0 && !seen.has(tid)
    )
    pending.value = []
    if (!id || !ids.length) return
    requested.value = [...requested.value, ...ids]
    for (let i = 0; i < ids.length; i += 100) {
      await fetchChunk(ids.slice(i, i + 100))
    }
  }

  const ensureLoaded = (tids: number[]) => {
    if (!id || !tids.length) return
    pending.value = [...pending.value, ...tids]
    if (flushScheduled.value) return
    flushScheduled.value = true
    queueMicrotask(() => {
      void flush()
    })
  }

  return {
    isFavorited: (tid: number) => states.value[tid]?.favorited ?? false,
    reactionKeysOf: (tid: number) => states.value[tid]?.reactions ?? [],
    ensureLoaded
  }
}
