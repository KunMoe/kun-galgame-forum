import { settle } from '#shared/utils/api/problem'
import type { TagNode } from '#shared/utils/api/schemas'

const loaded = new Map<number, Promise<TagNode[] | null>>()

export const useTagDescendants = (tagId: () => number) => {
  const api = useApiClient()
  const nodes = ref<TagNode[] | null>(null)
  const failed = ref(false)

  const load = (id: number) => {
    const hit = loaded.get(id)
    if (hit) {
      return hit
    }
    const pending = settle(
      api.GET('/tags/{tag_id}/descendants', {
        params: { path: { tag_id: String(id) }, query: { include_nsfw: true } }
      })
    ).then((result) => {
      if (!result.ok) {
        loaded.delete(id)
        return null
      }
      return result.data.items
    })
    loaded.set(id, pending)
    return pending
  }

  watch(
    tagId,
    async (id) => {
      nodes.value = null
      failed.value = false
      const result = await load(id)
      if (id !== tagId()) {
        return
      }
      nodes.value = result
      failed.value = result === null
    },
    { immediate: true }
  )

  return { nodes, failed }
}
