import { createHash } from 'node:crypto'
import { KUN_TOPIC_SECTION } from '../../app/constants/topic'
import type { components } from '../../shared/types/api/v1'
import {
  kunFeedUrl,
  type KunFeedFormat,
  type KunFeedUrlParams
} from '../../shared/utils/feedUrl'
import { parseFeedFile } from './kunFeed'
import type { SiteFeedParams } from './kunFeedSources'

type SectionKey = components['schemas']['SectionKey']
type Query = Record<string, unknown>

export type ParsedFeed<T> = {
  format: KunFeedFormat
  path: string
  params: T
  urlParams: KunFeedUrlParams
}

export type WorkFeedParams = { workId: string; includeNsfw: boolean }

export type UserFeedParams = {
  userId: string
  name: 'topic' | 'galgame'
  includeNsfw: boolean
}

const FEED_ID = /^[1-9][0-9]{0,18}$/
const SECTION_SHAPE = /^[a-z]-[a-z]{2,20}$/
const SOURCE_SHAPE = /^[a-z0-9_]{1,40}$/

const queryString = (query: Query, key: string): string | undefined => {
  const raw = query[key]
  const value = Array.isArray(raw) ? raw[0] : raw
  return value == null ? undefined : String(value)
}

const includeNsfwOf = (query: Query): boolean => {
  const value = queryString(query, 'include_nsfw')
  return value === '1' || value === 'true'
}

const nsfwUrlParams = (includeNsfw: boolean): KunFeedUrlParams =>
  includeNsfw ? { include_nsfw: true } : {}

// Nitro strips every non-word character from a custom cache key, so a key
// joined with ':' let '?lane=news&source=ymgal' and '?source=newsymgal' share
// one entry. The feed's own normalised URL, hashed, cannot collide.
export const feedCacheKey = (
  parsed: Omit<ParsedFeed<unknown>, 'params'> | null
): string =>
  parsed
    ? createHash('sha256')
        .update(kunFeedUrl('', parsed.path, parsed.format, parsed.urlParams))
        .digest('hex')
    : 'invalid'

export const parseSiteFeed = (
  file: string,
  query: Query
): ParsedFeed<SiteFeedParams> | null => {
  const parsed = parseFeedFile(file)
  if (!parsed) {
    return null
  }
  const { name, format } = parsed
  const includeNsfw = includeNsfwOf(query)

  switch (name) {
    case 'topic': {
      const raw = queryString(query, 'section')
      if (
        raw !== undefined &&
        (!SECTION_SHAPE.test(raw) || !(raw in KUN_TOPIC_SECTION))
      ) {
        return null
      }
      const section = raw as SectionKey | undefined
      return {
        format,
        path: '/rss/topic',
        params: { name, includeNsfw, section },
        urlParams: { ...nsfwUrlParams(includeNsfw), section }
      }
    }
    case 'galgame':
      return {
        format,
        path: '/rss/galgame',
        params: { name, includeNsfw },
        urlParams: nsfwUrlParams(includeNsfw)
      }
    case 'news': {
      const lane = queryString(query, 'lane')
      const source = queryString(query, 'source')
      if (lane !== undefined && lane !== 'news' && lane !== 'column') {
        return null
      }
      if (source !== undefined && !SOURCE_SHAPE.test(source)) {
        return null
      }
      return {
        format,
        path: '/rss/news',
        params: { name, lane, source },
        urlParams: { lane, source }
      }
    }
    case 'toolset':
      return { format, path: '/rss/toolset', params: { name }, urlParams: {} }
    default:
      return null
  }
}

export const parseWorkFeed = (
  file: string,
  query: Query
): ParsedFeed<WorkFeedParams> | null => {
  const parsed = parseFeedFile(file)
  if (!parsed || !FEED_ID.test(parsed.name)) {
    return null
  }
  const includeNsfw = includeNsfwOf(query)
  return {
    format: parsed.format,
    path: `/rss/galgame/${parsed.name}`,
    params: { workId: parsed.name, includeNsfw },
    urlParams: nsfwUrlParams(includeNsfw)
  }
}

export const parseUserFeed = (
  userId: string,
  file: string,
  query: Query
): ParsedFeed<UserFeedParams> | null => {
  const parsed = parseFeedFile(file)
  if (
    !FEED_ID.test(userId) ||
    !parsed ||
    (parsed.name !== 'topic' && parsed.name !== 'galgame')
  ) {
    return null
  }
  const includeNsfw = includeNsfwOf(query)
  return {
    format: parsed.format,
    path: `/rss/user/${userId}/${parsed.name}`,
    params: { userId, name: parsed.name, includeNsfw },
    urlParams: nsfwUrlParams(includeNsfw)
  }
}
