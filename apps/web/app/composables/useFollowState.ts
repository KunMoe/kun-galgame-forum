import { settle } from '#shared/utils/api/problem'
import type { UserFollowState } from '#shared/utils/api/schemas'

const FRESH_MS = 60_000

interface FollowEntry {
  state: UserFollowState
  at: number
}

const inflight = new Map<string, Promise<void>>()

export const useFollowState = (userId: MaybeRefOrGetter<number | string>) => {
  const api = useApiClient()
  const { id: currentUserId } = storeToRefs(usePersistUserStore())
  const entries = useState<Record<string, FollowEntry>>(
    'user-follow-state',
    () => ({})
  )
  const targetId = computed(() => String(toValue(userId)))
  const key = computed(() => `${currentUserId.value}:${targetId.value}`)
  const state = computed(() => entries.value[key.value]?.state ?? null)
  const busy = ref(false)

  const set = (next: UserFollowState) => {
    entries.value[key.value] = { state: next, at: Date.now() }
  }

  const fetchState = async (entryKey: string, id: string) => {
    const result = await settle(
      api.GET('/me/following/{user_id}', {
        params: { path: { user_id: id } }
      })
    )
    if (result.ok) {
      entries.value[entryKey] = { state: result.data, at: Date.now() }
    }
  }

  const load = async () => {
    if (!currentUserId.value) {
      return
    }
    const entryKey = key.value
    const cached = entries.value[entryKey]
    if (cached && Date.now() - cached.at < FRESH_MS) {
      return
    }
    const pending =
      inflight.get(entryKey) ??
      fetchState(entryKey, targetId.value).finally(() =>
        inflight.delete(entryKey)
      )
    inflight.set(entryKey, pending)
    await pending
  }

  const toggle = async (): Promise<-1 | 0 | 1> => {
    if (!currentUserId.value) {
      useAuthModal().open()
      return 0
    }
    const current = state.value
    if (!current || busy.value) {
      return 0
    }
    busy.value = true
    const path = { params: { path: { user_id: targetId.value } } }
    const result = await settle(
      current.is_following
        ? api.DELETE('/me/following/{user_id}', path)
        : api.PUT('/me/following/{user_id}', path)
    )
    busy.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return 0
    }
    set(result.data)
    if (result.data.is_following === current.is_following) {
      return 0
    }
    return result.data.is_following ? 1 : -1
  }

  return { state, busy, set, load, toggle }
}
