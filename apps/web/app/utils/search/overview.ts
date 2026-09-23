import type { ApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import type {
  CommentSearchHit,
  ReplySearchHit,
  TopicSummary,
  UserSearchHit,
  WallCommentSearchHit,
  WorkRef
} from '#shared/utils/api/schemas'
import {
  ENTITY_LIMIT_ALL,
  SEARCH_ENTITY_FAMILY_VALUES,
  fetchEntityFamilies
} from './entities'
import { fetchLanePage, type LanePage } from './lanes'

export interface SearchOverviewData {
  topics: TopicSummary[]
  works: WorkRef[]
  entities: SearchEntityGroup[]
  resources: SearchResultResource[]
  users: UserSearchHit[]
  replies: ReplySearchHit[]
  comments: CommentSearchHit[]
  wallComments: WallCommentSearchHit[]
  toolsets: SearchResultToolset[]
  totals: SearchOverviewTotals
  failed: boolean
}

const LIMIT = {
  topic: 8,
  galgame: 12,
  entity: ENTITY_LIMIT_ALL,
  resource: 6,
  user: 8,
  reply: 4,
  comment: 4,
  galcomment: 4,
  toolset: 4
} as const

export const loadSearchOverview = async (
  api: ApiClient,
  q: string,
  includeNsfw: boolean,
  preferOriginal: boolean
): Promise<SearchOverviewData> => {
  const lane = (type: SearchPagedType) =>
    fetchLanePage(api, type, q, 1, LIMIT[type], includeNsfw, {}, false)
  const [
    topics,
    works,
    resources,
    users,
    replies,
    comments,
    toolsets,
    entities,
    walls
  ] = await Promise.all([
    lane('topic'),
    lane('galgame'),
    lane('resource'),
    lane('user'),
    lane('reply'),
    lane('comment'),
    lane('toolset'),
    fetchEntityFamilies(
      api,
      q,
      1,
      LIMIT.entity,
      includeNsfw,
      preferOriginal,
      SEARCH_ENTITY_FAMILY_VALUES,
      false
    ),
    q.trim().length < 2
      ? Promise.resolve(null)
      : settle(
          api.GET('/search/wall-comments', {
            params: { query: { q, limit: LIMIT.galcomment } }
          })
        )
  ])

  const items = <T>(page: LanePage | null) => (page?.items ?? []) as T[]
  const totals: SearchOverviewTotals = {}
  const count = (type: SearchType, page: { total: number } | null) => {
    if (page) {
      totals[type] = page.total
    }
  }
  count('topic', topics)
  count('galgame', works)
  count('resource', resources)
  count('user', users)
  count('reply', replies)
  count('comment', comments)
  count('toolset', toolsets)
  const okEntities = entities.filter((group) => !group.failed)
  if (okEntities.length) {
    totals.entity = okEntities.reduce((sum, group) => sum + group.total, 0)
  }

  return {
    topics: items<TopicSummary>(topics),
    works: items<WorkRef>(works),
    resources: items<SearchResultResource>(resources),
    users: items<UserSearchHit>(users),
    replies: items<ReplySearchHit>(replies),
    comments: items<CommentSearchHit>(comments),
    toolsets: items<SearchResultToolset>(toolsets),
    entities,
    wallComments: walls?.ok ? walls.data.items : [],
    totals,
    failed:
      [
        topics,
        works,
        resources,
        users,
        replies,
        comments,
        toolsets
      ].every((page) => !page) &&
      !okEntities.length &&
      !walls?.ok
  }
}
