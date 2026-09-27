import type { ApiClient } from '../../shared/utils/api/client'
import { settle } from '../../shared/utils/api/problem'
import type {
  GalgameResource,
  UserTopicItem
} from '../../shared/utils/api/schemas'
import { catalogEntityName } from '../../shared/utils/catalogName'
import { deletedUserName } from '../../shared/utils/deletedUser'
import type { KunFeedResult } from './kunFeed'
import {
  resourceFeedItem,
  siteLogo,
  topicFeedItem,
  workCoverUrl
} from './kunFeedItems'
import {
  feedFailure,
  feedOk,
  fetchTopicDetails,
  nsfwQuery,
  topicDescription
} from './kunFeedSources'

export const loadWorkResourceFeed = async (
  api: ApiClient,
  baseUrl: string,
  params: { workId: string; includeNsfw: boolean }
): Promise<KunFeedResult> => {
  const list = await settle(
    api.GET('/works/{work_id}/resources', {
      params: {
        path: { work_id: params.workId },
        query: { state: 'valid', limit: 50 }
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
  const work = list.data.items[0]?.work
  const workName = work
    ? catalogEntityName(work).name
    : `Galgame ${params.workId}`
  const cover = work ? workCoverUrl(work, params.includeNsfw) : undefined
  return feedOk({
    title: `${workName} 的新资源 - 鲲 Galgame 论坛`,
    description: '鲲 Galgame 论坛上这部作品新发布的下载资源',
    link: `${baseUrl}/galgame/${params.workId}`,
    image: cover ?? siteLogo(baseUrl),
    items
  })
}

export const loadUserTopicFeed = async (
  api: ApiClient,
  baseUrl: string,
  params: { userId: string; includeNsfw: boolean }
): Promise<KunFeedResult> => {
  const list = await settle(
    api.GET('/users/{user_id}/topics', {
      params: {
        path: { user_id: params.userId },
        query: {
          relation: 'authored',
          limit: 20,
          ...nsfwQuery(params.includeNsfw)
        }
      }
    })
  )
  if (!list.ok) {
    return feedFailure(list.problem)
  }
  const topics = list.data.items
  const details = await fetchTopicDetails(api, topics)
  const items = topics.map((topic: UserTopicItem, index) => {
    const detail = details[index]
    const author = detail?.ok
      ? detail.data.author
      : { id: params.userId, name: `用户 ${params.userId}` }
    return topicFeedItem(baseUrl, topic, author, topicDescription(detail!))
  })
  const name = details[0]?.ok
    ? (details[0].data.author.name ?? deletedUserName)
    : `用户 ${params.userId}`
  return feedOk({
    title: `${name} 的话题 - 鲲 Galgame 论坛`,
    description: `${name} 在鲲 Galgame 论坛发布的话题`,
    link: `${baseUrl}/user/${params.userId}/info`,
    image: siteLogo(baseUrl),
    items
  })
}

export const loadUserGalgameFeed = async (
  api: ApiClient,
  baseUrl: string,
  params: { userId: string; includeNsfw: boolean }
): Promise<KunFeedResult> => {
  const list = await settle(
    api.GET('/users/{user_id}/galgame-resources', {
      params: {
        path: { user_id: params.userId },
        query: {
          relation: 'published',
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
  const first = list.data.items[0]
  const name = first
    ? (first.author.name ?? deletedUserName)
    : `用户 ${params.userId}`
  return feedOk({
    title: `${name} 发布的 Galgame 资源 - 鲲 Galgame 论坛`,
    description: `${name} 在鲲 Galgame 论坛发布的 Galgame 下载资源`,
    link: `${baseUrl}/user/${params.userId}/resource/valid`,
    image: siteLogo(baseUrl),
    items
  })
}
