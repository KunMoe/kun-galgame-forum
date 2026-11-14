import { settle } from '#shared/utils/api/problem'
import type { FollowingActivityItem } from '#shared/utils/api/schemas'

type ExpandableGroup = {
  id: string
  item_count: number
  items: FollowingActivityItem[]
}

export const useActivityGroupItems = (
  group: () => ExpandableGroup,
  includeNsfw: () => boolean,
  onGone: () => void
) => {
  const api = useApiClient()
  const expanded = ref<FollowingActivityItem[] | null>(null)
  const nextCursor = ref<string>()
  const loading = ref(false)

  const items = computed(() => expanded.value ?? group().items)
  const hasHidden = computed(
    () => expanded.value === null && group().item_count > group().items.length
  )

  const loadItems = async () => {
    if (loading.value) {
      return
    }
    loading.value = true
    const result = await settle(
      api.GET('/activity-groups/{group_id}/items', {
        params: {
          path: { group_id: group().id },
          query: {
            include_nsfw: includeNsfw(),
            limit: 20,
            ...(nextCursor.value ? { cursor: nextCursor.value } : {})
          }
        }
      })
    )
    loading.value = false
    if (!result.ok) {
      if (result.problem.status === 404) {
        onGone()
        return
      }
      reportProblem(result.problem)
      return
    }
    expanded.value = [...(expanded.value ?? []), ...result.data.items]
    nextCursor.value = result.data.next_cursor
  }

  return { items, hasHidden, nextCursor, loading, loadItems }
}
