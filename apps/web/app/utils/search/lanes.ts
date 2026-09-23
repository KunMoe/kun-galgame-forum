import type { ApiClient } from '#shared/utils/api/client'
import { settle, type ApiResult } from '#shared/utils/api/problem'

export interface LanePage {
  items: SearchResult[]
  total: number
  atLeast?: boolean
}

export type GalgameFilterQuery = Partial<
  Record<(typeof SEARCH_GALGAME_FILTER_KEYS)[number], string>
>

// The URL keeps the sort tokens it always had, so a link shared before the
// move still opens; the v1 face spells the direction out.
const WORKS_SORT = {
  relevance: 'relevance_desc',
  popularity: 'popularity_desc',
  updated: 'updated_desc',
  released_desc: 'released_desc',
  released_asc: 'released_asc'
} as const

const worksFilter = (filter: GalgameFilterQuery) => {
  const tagIDs = (filter.tag_ids ?? '')
    .split(',')
    .filter((id) => /^\d+$/.test(id))
  const sort =
    WORKS_SORT[(filter.sort ?? 'relevance') as keyof typeof WORKS_SORT]
  return {
    ...(filter.company_id ? { company_id: filter.company_id } : {}),
    ...(tagIDs.length ? { tag_ids: tagIDs } : {}),
    ...(filter.released_from ? { released_from: filter.released_from } : {}),
    ...(filter.released_to ? { released_to: filter.released_to } : {}),
    ...(sort ? { sort } : {})
  }
}

const unwrap = <
  T extends { items: unknown[]; total: number; total_relation: 'eq' | 'gte' }
>(
  result: ApiResult<T>,
  report: boolean
): LanePage | null => {
  if (!result.ok) {
    if (report) {
      reportProblem(result.problem)
    }
    return null
  }
  return {
    items: result.data.items as SearchResult[],
    total: result.data.total,
    atLeast: result.data.total_relation === 'gte'
  }
}

export const fetchLanePage = async (
  api: ApiClient,
  type: SearchPagedType,
  q: string,
  page: number,
  limit: number,
  includeNsfw: boolean,
  filter: GalgameFilterQuery = {},
  report = true
): Promise<LanePage | null> => {
  const paged = { q, page, limit }
  const shown = { ...paged, include_nsfw: includeNsfw }
  switch (type) {
    case 'topic':
      return unwrap(
        await settle(api.GET('/search/topics', { params: { query: shown } })),
        report
      )
    case 'reply':
      return unwrap(
        await settle(api.GET('/search/replies', { params: { query: shown } })),
        report
      )
    case 'comment':
      return unwrap(
        await settle(api.GET('/search/comments', { params: { query: shown } })),
        report
      )
    case 'user':
      return unwrap(
        await settle(api.GET('/search/users', { params: { query: paged } })),
        report
      )
    case 'galgame':
      return unwrap(
        await settle(
          api.GET('/search/works', {
            params: { query: { ...shown, ...worksFilter(filter) } }
          })
        ),
        report
      )
    case 'resource':
      return kunFetch<LanePage>('/search', {
        method: 'GET',
        query: { keywords: q, type: 'resource', page, limit }
      })
    case 'toolset':
      return kunFetch<LanePage>('/toolset', {
        method: 'GET',
        query: { query: q, page, limit }
      })
  }
}
