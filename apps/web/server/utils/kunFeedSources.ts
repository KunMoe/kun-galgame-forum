import type { ApiClient } from '../../shared/utils/api/client'
import type { ClientProblem } from '../../shared/utils/api/problem'
import { settle } from '../../shared/utils/api/problem'
import type {
  GalgameResource,
  TopicSummary
} from '../../shared/utils/api/schemas'
import type { components } from '../../shared/types/api/v1'
import { TOPIC_SECTION_OPTIONS } from '../../app/constants/topic'
import {
  KUN_GALGAME_TOOLSET_LANGUAGE_MAP,
  KUN_GALGAME_TOOLSET_PLATFORM_MAP,
  KUN_GALGAME_TOOLSET_TYPE_MAP
} from '../../app/constants/toolset'
import { documentPlainText } from '../../shared/utils/content/plainText'
import { deletedUserName } from '../../shared/utils/deletedUser'
import { truncateRunes } from '../../shared/utils/format'
import type { KunFeedChannel, KunFeedResult } from './kunFeed'
import {
  authorEntry,
  FEED_DESCRIPTION_RUNES,
  resourceFeedItem,
  siteLogo,
  topicFeedItem
} from './kunFeedItems'

type SectionKey = components['schemas']['SectionKey']
type NewsLane = components['schemas']['NewsLane']
type NewsItem = components['schemas']['NewsItem']
type NewsSource = components['schemas']['NewsSource']
type ToolsetSummary = components['schemas']['ToolsetSummary']

export const feedStatus = (problem: ClientProblem): 404 | 502 =>
  problem.status >= 400 && problem.status < 500 ? 404 : 502

export const feedFailure = (problem: ClientProblem): KunFeedResult => ({
  ok: false,
  status: feedStatus(problem)
})

export const nsfwQuery = (includeNsfw: boolean) =>
  includeNsfw ? { include_nsfw: true as const } : {}

export const feedOk = (channel: KunFeedChannel): KunFeedResult => ({
  ok: true,
  channel
})

export const fetchTopicDetails = (api: ApiClient, topics: { id: string }[]) =>
  Promise.all(
    topics.map((topic) =>
      settle(
        api.GET('/topics/{topic_id}', {
          params: { path: { topic_id: topic.id } }
        })
      )
    )
  )

export const topicDescription = (
  detail: Awaited<ReturnType<typeof fetchTopicDetails>>[number]
) =>
  detail.ok
    ? truncateRunes(
        documentPlainText(detail.data.content, deletedUserName),
        FEED_DESCRIPTION_RUNES
      )
    : ''

export const loadTopicFeed = async (
  api: ApiClient,
  baseUrl: string,
  params: { includeNsfw: boolean; section?: SectionKey }
): Promise<KunFeedResult> => {
  const list = await settle(
    api.GET('/topics', {
      params: {
        query: {
          sort: 'created_desc',
          limit: 20,
          ...nsfwQuery(params.includeNsfw),
          ...(params.section ? { section: params.section } : {})
        }
      }
    })
  )
  if (!list.ok) {
    return feedFailure(list.problem)
  }
  const details = await fetchTopicDetails(api, list.data.items)
  const items = list.data.items.map((topic: TopicSummary, index) =>
    topicFeedItem(
      baseUrl,
      topic,
      topic.author,
      topicDescription(details[index]!)
    )
  )
  const sectionLabel = TOPIC_SECTION_OPTIONS.find(
    (option) => option.value === params.section
  )?.label
  return feedOk({
    title: sectionLabel
      ? `鲲 Galgame 论坛 - ${sectionLabel} 分区新话题`
      : '鲲 Galgame 论坛 - 新话题',
    description: '鲲 Galgame 论坛最新发布的话题',
    link: params.section
      ? `${baseUrl}/section/${params.section}`
      : `${baseUrl}/topic`,
    image: siteLogo(baseUrl),
    items
  })
}

export const loadGalgameFeed = async (
  api: ApiClient,
  baseUrl: string,
  params: { includeNsfw: boolean }
): Promise<KunFeedResult> => {
  const list = await settle(
    api.GET('/galgame-resources', {
      params: {
        query: {
          sort: 'created_desc',
          limit: 50,
          ...nsfwQuery(params.includeNsfw)
        }
      }
    })
  )
  if (!list.ok) {
    return feedFailure(list.problem)
  }
  const items = list.data.items.flatMap((resource: GalgameResource) => {
    const item = resourceFeedItem(baseUrl, resource, params.includeNsfw)
    return item ? [item] : []
  })
  return feedOk({
    title: '鲲 Galgame 论坛 - Galgame 新资源',
    description: '鲲 Galgame 论坛最新发布的 Galgame 下载资源',
    link: `${baseUrl}/galgame/resource`,
    image: siteLogo(baseUrl),
    items
  })
}

export const loadNewsFeed = async (
  api: ApiClient,
  baseUrl: string,
  params: { lane?: NewsLane; source?: string }
): Promise<KunFeedResult> => {
  const [itemsResult, sourcesResult] = await Promise.all([
    settle(
      api.GET('/news-items', {
        params: {
          query: {
            limit: 30,
            ...(params.lane ? { lane: params.lane } : {}),
            ...(params.source ? { news_source: params.source } : {})
          }
        }
      })
    ),
    settle(api.GET('/news-sources', {}))
  ])
  if (!sourcesResult.ok) {
    return feedFailure(sourcesResult.problem)
  }
  if (!itemsResult.ok) {
    return feedFailure(itemsResult.problem)
  }
  const sources = new Map(
    sourcesResult.data.items.map((source: NewsSource) => [source.key, source])
  )
  const items = itemsResult.data.items.flatMap((item: NewsItem) => {
    const source = sources.get(item.news_source)
    if (!source) {
      return []
    }
    const preview = item.preview.trim()
    const attribution = source.attribution.trim()
    const description = [preview, attribution || undefined]
      .filter((part): part is string => Boolean(part))
      .join('\n\n')
    return [
      {
        link: item.source_url,
        title: item.title.trim(),
        date: new Date(item.published_at),
        description,
        author: [
          {
            name: source.display_name,
            ...(source.homepage_url ? { link: source.homepage_url } : {})
          }
        ],
        category: [{ name: item.lane === 'news' ? '情报' : '专栏' }]
      }
    ]
  })
  return feedOk({
    title: '鲲 Galgame 论坛 - Gal 情报',
    description: '鲲 Galgame 论坛转载的 Galgame 情报与专栏, 版权归原作者所有',
    link: `${baseUrl}/news`,
    image: siteLogo(baseUrl),
    items
  })
}

export const loadToolsetFeed = async (
  api: ApiClient,
  baseUrl: string
): Promise<KunFeedResult> => {
  const list = await settle(
    api.GET('/toolsets', {
      params: { query: { sort: 'created_desc', limit: 20 } }
    })
  )
  if (!list.ok) {
    return feedFailure(list.problem)
  }
  const items = list.data.items.map((toolset: ToolsetSummary) => {
    const aliases = toolset.aliases
    const description = [
      `类型: ${KUN_GALGAME_TOOLSET_TYPE_MAP[toolset.toolset_type] ?? toolset.toolset_type}`,
      `平台: ${KUN_GALGAME_TOOLSET_PLATFORM_MAP[toolset.platform] ?? toolset.platform}`,
      `语言: ${KUN_GALGAME_TOOLSET_LANGUAGE_MAP[toolset.interface_language] ?? toolset.interface_language}`,
      ...(aliases.length ? [`别名: ${aliases.join(' / ')}`] : [])
    ].join(' | ')
    return {
      link: `${baseUrl}/toolset/${toolset.id}`,
      title: toolset.title,
      date: new Date(toolset.created_at),
      description,
      author: [authorEntry(toolset.author, baseUrl)]
    }
  })
  return feedOk({
    title: '鲲 Galgame 论坛 - 新工具',
    description: '鲲 Galgame 论坛最新收录的 Galgame 工具',
    link: `${baseUrl}/toolset`,
    image: siteLogo(baseUrl),
    items
  })
}

export type SiteFeedParams =
  | { name: 'topic'; includeNsfw: boolean; section?: SectionKey }
  | { name: 'galgame'; includeNsfw: boolean }
  | { name: 'news'; lane?: NewsLane; source?: string }
  | { name: 'toolset' }

export const loadSiteFeed = (
  api: ApiClient,
  baseUrl: string,
  params: SiteFeedParams
): Promise<KunFeedResult> => {
  switch (params.name) {
    case 'topic':
      return loadTopicFeed(api, baseUrl, params)
    case 'galgame':
      return loadGalgameFeed(api, baseUrl, params)
    case 'news':
      return loadNewsFeed(api, baseUrl, params)
    case 'toolset':
      return loadToolsetFeed(api, baseUrl)
  }
}
