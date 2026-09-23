import { settle } from '#shared/utils/api/problem'

export const useMyQuizInteractions = () => {
  const { id } = usePersistUserStore()
  const api = useApiClient()
  const states = useState<Record<string, boolean>>(
    'my-quiz-favorited',
    () => ({})
  )
  const requested = useState<string[]>('my-quiz-states-requested', () => [])
  const pending = useState<string[]>('my-quiz-states-pending', () => [])
  const flushScheduled = useState<boolean>(
    'my-quiz-states-flush-scheduled',
    () => false
  )

  const fetchChunk = async (chunk: string[]) => {
    const res = await settle(
      api.GET('/me/quiz-states', {
        params: { query: { quiz_ids: chunk } }
      })
    )
    if (!res.ok) {
      const failed = new Set(chunk)
      requested.value = requested.value.filter((qid) => !failed.has(qid))
      return
    }
    const next = { ...states.value }
    for (const item of res.data.items) {
      next[item.quiz_id] = item.has_favorited
    }
    states.value = next
  }

  const flush = async () => {
    flushScheduled.value = false
    const seen = new Set(requested.value)
    const ids = [...new Set(pending.value)].filter(
      (qid) => qid.length > 0 && !seen.has(qid)
    )
    pending.value = []
    if (!id || !ids.length) return
    requested.value = [...requested.value, ...ids]
    for (let i = 0; i < ids.length; i += 100) {
      await fetchChunk(ids.slice(i, i + 100))
    }
  }

  const ensureLoaded = (quizIds: Array<string | number>) => {
    if (!id || !quizIds.length) return
    pending.value = [...pending.value, ...quizIds.map(String)]
    if (flushScheduled.value) return
    flushScheduled.value = true
    queueMicrotask(() => {
      void flush()
    })
  }

  const setFavorited = (quizId: string | number, isFav: boolean) => {
    states.value = { ...states.value, [String(quizId)]: isFav }
  }

  return {
    isFavorited: (quizId: string | number) =>
      states.value[String(quizId)] ?? false,
    setFavorited,
    ensureLoaded
  }
}
