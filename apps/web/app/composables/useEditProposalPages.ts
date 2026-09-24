import { settle } from '#shared/utils/api/problem'
import type { EditProposalSummary } from '~/utils/galgame/editAdapt'

type Page = { items: EditProposalSummary[]; next_cursor?: string }

export const useEditProposalPages = (
  first: Ref<Page | undefined> | ComputedRef<Page | undefined>,
  fetchPage: (cursor: string) => Promise<{
    data?: Page
    error?: unknown
    response: Response
  }>
) => {
  const extra = ref<EditProposalSummary[]>([])
  const cursor = ref<string | undefined>()
  const loadingMore = ref(false)

  watch(
    first,
    (page) => {
      extra.value = []
      cursor.value = page?.next_cursor
    },
    { immediate: true }
  )

  const items = computed(() => [...(first.value?.items ?? []), ...extra.value])

  const loadMore = async () => {
    if (!cursor.value || loadingMore.value) {
      return
    }
    loadingMore.value = true
    const result = await settle(fetchPage(cursor.value))
    loadingMore.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    extra.value.push(...result.data.items)
    cursor.value = result.data.next_cursor
  }

  return { items, hasMore: computed(() => !!cursor.value), loadingMore, loadMore }
}
