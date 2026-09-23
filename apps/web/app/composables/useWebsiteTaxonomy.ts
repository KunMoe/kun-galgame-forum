import type {
  WebsiteCategory,
  WebsiteTag,
  WebsiteTagGroup
} from '#shared/utils/api/schemas'

type PageResult<T> = {
  data?: { items: T[]; next_cursor?: string }
  error?: unknown
  response: Response
}

export const collectPages = async <T>(
  fetchPage: (cursor: string | undefined) => Promise<PageResult<T>>
): Promise<{ data?: T[]; error?: unknown; response: Response }> => {
  const items: T[] = []
  let cursor: string | undefined
  for (;;) {
    const page = await fetchPage(cursor)
    if (!page.data) {
      return { error: page.error, response: page.response }
    }
    items.push(...page.data.items)
    if (!page.data.next_cursor) {
      return { data: items, response: page.response }
    }
    cursor = page.data.next_cursor
  }
}

export const useWebsiteCategories = () =>
  useApi<WebsiteCategory[]>('website-categories', (api) =>
    collectPages((cursor) =>
      api.GET('/website-categories', {
        params: { query: { limit: 100, ...(cursor ? { cursor } : {}) } }
      })
    )
  )

export const useWebsiteTagGroups = () =>
  useApi<WebsiteTagGroup[]>('website-tag-groups', (api) =>
    collectPages((cursor) =>
      api.GET('/website-tag-groups', {
        params: { query: { limit: 100, ...(cursor ? { cursor } : {}) } }
      })
    )
  )

export const useWebsiteTags = () =>
  useApi<WebsiteTag[]>('website-tags', (api) =>
    collectPages((cursor) =>
      api.GET('/website-tags', {
        params: { query: { limit: 100, ...(cursor ? { cursor } : {}) } }
      })
    )
  )
