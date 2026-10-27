import { settle, type ApiResult } from '#shared/utils/api/problem'
import type { FriendLink } from '#shared/utils/api/schemas'

const LINK_PAGE_LIMIT = 100
const LINK_MAX_PAGES = 20

export const useFriendLinks = () => {
  const list = useKunApi(
    'friend-links:all',
    async (api, { signal }): Promise<ApiResult<FriendLink[]>> => {
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
  const links = computed(() => list.data.value ?? [])
  const state = { links, pending: list.pending, refresh: list.refresh }
  return Object.assign(
    Promise.resolve(list).then(() => state),
    state
  )
}
