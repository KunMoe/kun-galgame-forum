import type { H3Event } from 'h3'
import { createApiClient, type ApiClient } from '../../shared/utils/api/client'
import { kunFeedUrl } from '../../shared/utils/feedUrl'
import { renderKunFeed, type KunFeedResult } from './kunFeed'
import {
  loadUserGalgameFeed,
  loadUserTopicFeed,
  loadWorkResourceFeed
} from './kunFeedEntitySources'
import {
  feedCacheKey,
  parseSiteFeed,
  parseUserFeed,
  parseWorkFeed,
  type ParsedFeed,
  type UserFeedParams
} from './kunFeedParams'
import { loadSiteFeed } from './kunFeedSources'

const defineKunFeedRoute = <T>(options: {
  name: string
  parse: (event: H3Event) => ParsedFeed<T> | null
  load: (api: ApiClient, baseUrl: string, params: T) => Promise<KunFeedResult>
}) =>
  defineCachedEventHandler(
    async (event) => {
      const parsed = options.parse(event)
      if (!parsed) {
        throw createError({ statusCode: 404 })
      }
      const config = useRuntimeConfig()
      const baseUrl = config.public.KUN_GALGAME_URL || ''
      const api = createApiClient({
        origin: config.apiBaseUrl,
        timeoutMs: 10000
      })
      const result = await options.load(api, baseUrl, parsed.params)
      if (!result.ok) {
        throw createError({ statusCode: result.status })
      }
      const link = (format: 'xml' | 'atom' | 'json') =>
        kunFeedUrl(baseUrl, parsed.path, format, parsed.urlParams)
      const rendered = renderKunFeed(result.channel, parsed.format, baseUrl, {
        rss: link('xml'),
        atom: link('atom'),
        json: link('json')
      })
      setHeader(event, 'Content-Type', rendered.contentType)
      return rendered.body
    },
    {
      name: options.name,
      base: 'rss',
      getKey: (event) => feedCacheKey(options.parse(event)),
      swr: true,
      maxAge: 60 * 5,
      staleMaxAge: 60 * 60 * 24 * 7
    }
  )

const loadUserFeed = (
  api: ApiClient,
  baseUrl: string,
  params: UserFeedParams
): Promise<KunFeedResult> =>
  params.name === 'topic'
    ? loadUserTopicFeed(api, baseUrl, params)
    : loadUserGalgameFeed(api, baseUrl, params)

export const siteFeedHandler = defineKunFeedRoute({
  name: 'rss-site',
  parse: (event) =>
    parseSiteFeed(getRouterParam(event, 'feed') ?? '', getQuery(event)),
  load: loadSiteFeed
})

export const workFeedHandler = defineKunFeedRoute({
  name: 'rss-work',
  parse: (event) =>
    parseWorkFeed(getRouterParam(event, 'work') ?? '', getQuery(event)),
  load: loadWorkResourceFeed
})

export const userFeedHandler = defineKunFeedRoute({
  name: 'rss-user',
  parse: (event) =>
    parseUserFeed(
      getRouterParam(event, 'user') ?? '',
      getRouterParam(event, 'feed') ?? '',
      getQuery(event)
    ),
  load: loadUserFeed
})
