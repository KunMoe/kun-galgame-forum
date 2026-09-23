import { settle, type ApiResult } from '#shared/utils/api/problem'
import type { FriendLink } from '#shared/utils/api/schemas'

const LINK_PAGE_LIMIT = 100
const LINK_MAX_PAGES = 20

export const useFriendLinks = () => {
  const api = useApiClient()
  const asyncData = useAsyncData(
    'api:friend-links:all',
    async (_app, { signal }): Promise<ApiResult<FriendLink[]>> => {
      const items: FriendLink[] = []
      let cursor: string | undefined
      for (let page = 0; page < LINK_MAX_PAGES; page++) {
        const result = await settle(
          api.GET('/friend-links', {
            params: {
              query: { limit: LINK_PAGE_LIMIT, ...(cursor ? { cursor } : {}) }
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
  )
  const links = computed(() =>
    asyncData.data.value?.ok ? asyncData.data.value.data : []
  )
  return Object.assign(
    Promise.resolve(asyncData).then(() => ({
      links,
      refresh: asyncData.refresh
    })),
    { links, refresh: asyncData.refresh }
  )
}
