import { createApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import { catalogEntityName } from '#shared/utils/catalogName'
import { useKunFeed } from '../../utils/kunFeed'

const RSS_WORK_COUNT = 20

export default defineCachedEventHandler(
  async (event): Promise<string> => {
    const config = useRuntimeConfig()
    const baseUrl = config.public.KUN_GALGAME_URL || ''
    const feed = useKunFeed(baseUrl, 'galgame')
    const api = createApiClient({ origin: config.apiBaseUrl, timeoutMs: 10000 })

    const list = await settle(
      api.GET('/works', {
        params: {
          query: { sort: 'resource_updated_desc', limit: RSS_WORK_COUNT }
        }
      })
    )
    if (!list.ok) {
      throw createError({
        statusCode: 502,
        statusMessage: 'work list unavailable'
      })
    }

    for (const work of list.data.items) {
      if (!work.resource_updated_at) {
        continue
      }
      const { name } = catalogEntityName(work)
      feed.addItem({
        link: `${baseUrl}/galgame/${work.id}`,
        title: name,
        date: new Date(work.resource_updated_at),
        description: '',
        image: work.banner?.url ?? work.cover?.url
      })
    }

    setHeader(event, 'Content-Type', 'application/xml')
    return feed.rss2()
  },
  {
    name: 'rss-galgame',
    getKey: () => 'all',
    swr: true,
    maxAge: 60 * 5,
    staleMaxAge: 60 * 60 * 24 * 7
  }
)
