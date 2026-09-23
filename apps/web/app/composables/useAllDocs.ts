import type { ApiClient } from '#shared/utils/api/client'
import { settle, type ApiResult } from '#shared/utils/api/problem'
import type { DocSort, DocSummary } from '#shared/utils/api/schemas'

const DOC_PAGE_LIMIT = 100
const DOC_MAX_PAGES = 20

export const fetchAllDocs = async (
  api: ApiClient,
  sort: DocSort,
  signal?: AbortSignal
): Promise<ApiResult<DocSummary[]>> => {
  const items: DocSummary[] = []
  let cursor: string | undefined
  for (let page = 0; page < DOC_MAX_PAGES; page++) {
    const result = await settle(
      api.GET('/docs', {
        params: {
          query: { sort, limit: DOC_PAGE_LIMIT, ...(cursor ? { cursor } : {}) }
        },
        signal
      })
    )
    if (!result.ok) {
      return result
    }
    items.push(...result.data.items)
    cursor = result.data.next_cursor
    if (!cursor) {
      break
    }
  }
  return { ok: true, data: items }
}

export const useAllDocs = (sort: DocSort) => {
  const api = useApiClient()
  const asyncData = useAsyncData(`api:docs:all:${sort}`, (_app, { signal }) =>
    fetchAllDocs(api, sort, signal)
  )
  const docs = computed(() =>
    asyncData.data.value?.ok ? asyncData.data.value.data : []
  )
  return Object.assign(
    Promise.resolve(asyncData).then(() => ({
      docs,
      refresh: asyncData.refresh
    })),
    { docs, refresh: asyncData.refresh }
  )
}
