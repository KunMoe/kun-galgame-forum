import { settle } from '#shared/utils/api/problem'
import type { UserProfile } from '#shared/utils/api/schemas'

const FRESH_MS = 60_000

type UserCardEntry =
  | { state: 'ready'; profile: UserProfile; at: number }
  | { state: 'missing'; at: number }

export const useUserCard = (userId: MaybeRefOrGetter<number | string>) => {
  const api = useApiClient()
  const entries = useState<Record<string, UserCardEntry>>(
    'user-card',
    () => ({})
  )
  const id = computed(() => String(toValue(userId)))
  const entry = computed(() => entries.value[id.value])
  const failed = ref(false)

  const load = async () => {
    const cached = entry.value
    if (cached && Date.now() - cached.at < FRESH_MS) {
      return
    }
    failed.value = false
    const key = id.value
    const res = await settle(
      api.GET('/users/{user_id}', { params: { path: { user_id: key } } })
    )
    if (res.ok) {
      entries.value[key] = { state: 'ready', profile: res.data, at: Date.now() }
    } else if (res.problem.status === 404) {
      entries.value[key] = { state: 'missing', at: Date.now() }
    } else {
      failed.value = !cached
    }
  }

  const adjustFollowers = (delta: number) => {
    const cached = entry.value
    if (cached?.state !== 'ready') {
      return
    }
    const { counts } = cached.profile
    if (counts.follower_count !== null) {
      counts.follower_count += delta
    }
  }

  return { entry, failed, load, adjustFollowers }
}
