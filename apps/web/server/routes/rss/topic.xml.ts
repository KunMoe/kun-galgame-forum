import { createApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import { documentPlainText } from '#shared/utils/content/plainText'
import { deletedUserName } from '#shared/utils/deletedUser'
import { truncateRunes } from '#shared/utils/format'
import { useKunFeed } from '../../utils/kunFeed'

const RSS_TOPIC_COUNT = 10
const DESCRIPTION_RUNES = 233

export default defineCachedEventHandler(
  async (event): Promise<string> => {
    const config = useRuntimeConfig()
    const baseUrl = config.public.KUN_GALGAME_URL || ''
    const feed = useKunFeed(baseUrl, 'topic')
    const api = createApiClient({ origin: config.apiBaseUrl, timeoutMs: 10000 })

    const list = await settle(
      api.GET('/topics', {
        params: { query: { sort: 'created_desc', limit: RSS_TOPIC_COUNT } }
      })
    )
    if (!list.ok) {
      throw createError({
        statusCode: 502,
        statusMessage: 'topic list unavailable'
      })
    }

    const details = await Promise.all(
      list.data.items.map((topic) =>
        settle(
          api.GET('/topics/{topic_id}', {
            params: { path: { topic_id: topic.id } }
          })
        )
      )
    )

    list.data.items.forEach((topic, i) => {
      const detail = details[i]
      const text = detail?.ok
        ? documentPlainText(detail.data.content, deletedUserName)
        : ''
      feed.addItem({
        link: `${baseUrl}/topic/${topic.id}`,
        title: topic.title,
        date: new Date(topic.created_at),
        description: truncateRunes(text, DESCRIPTION_RUNES),
        author: [
          {
            name: topic.author.name ?? deletedUserName,
            link: `${baseUrl}/user/${topic.author.id}/info`
          }
        ]
      })
    })

    setHeader(event, 'Content-Type', 'application/xml')
    return feed.rss2()
  },
  {
    name: 'rss-topic',
    getKey: () => 'all',
    swr: true,
    maxAge: 60 * 5,
    staleMaxAge: 60 * 60 * 24 * 7
  }
)
