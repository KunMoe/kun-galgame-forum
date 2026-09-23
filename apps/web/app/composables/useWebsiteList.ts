import type { WebsiteSummary } from '#shared/utils/api/schemas'

type WebsiteFilter = {
  website_category_id?: string
  website_tag_id?: string
}

export const byScore = (a: WebsiteSummary, b: WebsiteSummary) =>
  b.score - a.score || Number(b.id) - Number(a.id)

export const useWebsiteList = (
  filter: MaybeRefOrGetter<WebsiteFilter> = {}
) => {
  const { allowsNsfw } = useContentStance()
  return useApi<WebsiteSummary[]>(
    () =>
      `websites:${JSON.stringify(toValue(filter))}:${allowsNsfw.value ? 'nsfw' : 'sfw'}`,
    (api) =>
      collectPages((cursor) =>
        api.GET('/websites', {
          params: {
            query: {
              limit: 100,
              include_nsfw: allowsNsfw.value,
              ...toValue(filter),
              ...(cursor ? { cursor } : {})
            }
          }
        })
      )
  )
}
