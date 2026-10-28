import { settle } from '#shared/utils/api/problem'
import type { EntityTarget } from '#shared/utils/api/schemas'
import type { EntityRefType } from '~/utils/entityUrl'

const LIMIT = 12

export const ENTITY_PREVIEW_ORDER: EntityRefType[] = [
  'topic',
  'work',
  'galgame_resource',
  'user',
  'character',
  'company',
  'tag',
  'series',
  'credit_name',
  'rating'
]

export const entityPreviewQuery = (
  refs: { type: EntityRefType; id: string }[]
): Record<string, string[]> => {
  const query: Record<string, string[]> = {}
  for (const r of refs) {
    ;(query[`${r.type}_ids`] ??= []).push(r.id)
  }
  return query
}

export const orderForPreview = <T extends { type: EntityRefType }>(refs: T[]) =>
  ENTITY_PREVIEW_ORDER.flatMap((type) => refs.filter((r) => r.type === type))

export const useEntitySearch = () => {
  const api = useApiClient()
  const { allowsNsfw } = useContentStance()

  const ids = <T extends { items: { id: string }[] }>(r: {
    ok: boolean
    data?: T
  }): string[] => (r.ok && r.data ? r.data.items.map((i) => i.id) : [])

  const searchIds = async (
    type: EntityRefType,
    query: string,
    signal: AbortSignal
  ): Promise<string[]> => {
    const q = query.trim()
    if (!q) {
      return []
    }
    const include_nsfw = allowsNsfw.value
    const opts = { signal }
    switch (type) {
      case 'work':
        return ids(
          await settle(
            api.GET('/search/works', {
              params: { query: { q, limit: LIMIT, include_nsfw } },
              ...opts
            })
          )
        )
      case 'character':
        return ids(
          await settle(
            api.GET('/characters', {
              params: { query: { q, limit: LIMIT, include_nsfw } },
              ...opts
            })
          )
        )
      case 'company':
        return ids(
          await settle(
            api.GET('/companies', {
              params: { query: { q, limit: LIMIT } },
              ...opts
            })
          )
        )
      case 'tag':
        return ids(
          await settle(
            api.GET('/tags', {
              params: { query: { q, limit: LIMIT, include_nsfw } },
              ...opts
            })
          )
        )
      case 'series':
        return ids(
          await settle(
            api.GET('/series', {
              params: { query: { q, limit: LIMIT, include_nsfw } },
              ...opts
            })
          )
        )
      case 'credit_name':
        return ids(
          await settle(
            api.GET('/credit-names', {
              params: { query: { q, limit: LIMIT } },
              ...opts
            })
          )
        )
      case 'topic':
        return ids(
          await settle(
            api.GET('/search/topics', {
              params: { query: { q, limit: LIMIT, include_nsfw } },
              ...opts
            })
          )
        )
      case 'galgame_resource':
        return ids(
          await settle(
            api.GET('/galgame-resources', {
              params: { query: { q, limit: LIMIT, include_nsfw } },
              ...opts
            })
          )
        )
      case 'user':
        return ids(
          await settle(
            api.GET('/users', {
              params: { query: { q, limit: LIMIT } },
              ...opts
            })
          )
        )
      case 'rating':
        return []
    }
  }

  const ratingIdsOfWork = async (
    workId: string,
    signal: AbortSignal
  ): Promise<string[]> =>
    ids(
      await settle(
        api.GET('/ratings', {
          params: {
            query: {
              work_id: workId,
              sort: 'view_desc',
              limit: LIMIT,
              include_nsfw: allowsNsfw.value
            }
          },
          signal
        })
      )
    )

  const previews = async (
    refs: { type: EntityRefType; id: string }[],
    signal: AbortSignal
  ): Promise<EntityTarget[]> => {
    if (!refs.length) {
      return []
    }
    const r = await settle(
      api.GET('/entity-previews', {
        params: { query: entityPreviewQuery(orderForPreview(refs)) },
        signal
      })
    )
    return r.ok ? r.data.items : []
  }

  return { searchIds, ratingIdsOfWork, previews }
}
