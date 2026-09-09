import { useRouteQuery } from '@vueuse/router'
import type {
  KunGalgameResourceTypeOptions,
  KunGalgameResourceLanguageOptions,
  KunGalgameResourcePlatformOptions
} from '~/constants/galgame'

type SortField =
  | 'popularity'
  | 'time'
  | 'created'
  | 'view'
  | 'view_1d'
  | 'view_7d'
  | 'view_30d'
  | 'release_date'
  | 'rating'

export const GALGAME_FILTER_QUERY_KEYS = [
  'page',
  'type',
  'language',
  'platform',
  'gameType',
  'sortField',
  'sortOrder',
  'releasedFrom',
  'releasedTo',
  'releasedMonths',
  'collectedFrom',
  'collectedTo',
  'collectedMonths',
  'includeProviders',
  'excludeOnlyProviders',
  'minRatingCount',
  'minRating'
] as const

// popularity is a catalog sort and the local list cannot answer it, so only the
// library page may ask for it as its default.
export const useGalgameFilters = (defaultSortField: SortField = 'time') => {
  const opts = { mode: 'replace' as const }

  const page = useRouteQuery('page', 1, { ...opts, transform: Number })

  const type = useRouteQuery<KunGalgameResourceTypeOptions>('type', 'all', opts)
  const language = useRouteQuery<KunGalgameResourceLanguageOptions>(
    'language',
    'all',
    opts
  )
  const platform = useRouteQuery<KunGalgameResourcePlatformOptions>(
    'platform',
    'all',
    opts
  )
  const gameType = useRouteQuery<string>('gameType', 'all', opts)
  const sortField = useRouteQuery<SortField>(
    'sortField',
    defaultSortField,
    opts
  )
  const sortOrder = useRouteQuery<KunOrder>('sortOrder', 'desc', opts)

  const releasedFrom = useRouteQuery<string>('releasedFrom', '', opts)
  const releasedTo = useRouteQuery<string>('releasedTo', '', opts)
  const releasedMonths = useRouteQuery<string>('releasedMonths', '', opts)

  const collectedFrom = useRouteQuery<string>('collectedFrom', '', opts)
  const collectedTo = useRouteQuery<string>('collectedTo', '', opts)
  const collectedMonths = useRouteQuery<string>('collectedMonths', '', opts)

  const includeProviders = useRouteQuery<string>('includeProviders', '', opts)
  const excludeOnlyProviders = useRouteQuery<string>(
    'excludeOnlyProviders',
    '',
    opts
  )

  const minRatingCount = useRouteQuery('minRatingCount', 0, {
    ...opts,
    transform: Number
  })
  const minRating = useRouteQuery('minRating', 0, {
    ...opts,
    transform: Number
  })

  const limit = 24

  return {
    page,
    limit,
    type,
    language,
    platform,
    gameType,
    sortField,
    sortOrder,
    releasedFrom,
    releasedTo,
    releasedMonths,
    collectedFrom,
    collectedTo,
    collectedMonths,
    includeProviders,
    excludeOnlyProviders,
    minRatingCount,
    minRating
  }
}
