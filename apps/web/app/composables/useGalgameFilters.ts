import { useRouteQuery } from '@vueuse/router'
import type { ListWorksQuery } from '#shared/utils/api/schemas'
import {
  LANGUAGE_OPTIONS,
  LEGACY_RESOURCE_LANGUAGE,
  LEGACY_RESOURCE_PLATFORM,
  LEGACY_RESOURCE_TYPE,
  PLATFORM_OPTIONS,
  RESOURCE_TYPE_OPTIONS,
  axisKey
} from '#shared/utils/galgameResourceVocab'
import {
  PROVIDER_KEY_OPTIONS,
  type ProviderKey
} from '~/constants/galgameResource'

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

const BROWSE_SORT_FIELDS = new Set([
  'resource_updated',
  'created',
  'view',
  'view_1d',
  'view_7d',
  'view_30d',
  'release_date',
  'rating'
])

const GAME_TYPES = new Set<NonNullable<ListWorksQuery['game_type']>>([
  'ba_saku',
  'plot',
  'moe',
  'daily',
  'uncategorized'
])

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

const mappedAxis = (
  key: string,
  legacy: Record<string, string>,
  options: { value: string }[]
) => {
  const raw = useRouteQuery<string>(key, '', { mode: 'replace' })
  return computed({
    get: () => axisKey(raw.value, legacy, options) ?? '',
    set: (value: string) => {
      raw.value = value
    }
  })
}

const csvInts = (csv: string, min: number, max: number) => {
  const values = csv
    .split(',')
    .map(Number)
    .filter((n) => Number.isInteger(n) && n >= min && n <= max)
  return values.length ? values : undefined
}

const csvProviders = (
  csv: string
): NonNullable<ListWorksQuery['resource_providers']> | undefined => {
  const values = csv
    .split(',')
    .filter((key): key is ProviderKey =>
      (PROVIDER_KEY_OPTIONS as readonly string[]).includes(key)
    )
  return values.length ? values : undefined
}

export const browseSortToken = (
  field: string,
  order: string
): NonNullable<ListWorksQuery['sort']> => {
  const key =
    field === 'time'
      ? 'resource_updated'
      : field === 'views'
        ? 'view'
        : field
  const dir = order === 'asc' ? 'asc' : 'desc'
  const token = `${key}_${dir}`
  return BROWSE_SORT_FIELDS.has(key)
    ? (token as NonNullable<ListWorksQuery['sort']>)
    : 'resource_updated_desc'
}

export const useGalgameFilters = (defaultSortField: SortField = 'time') => {
  const opts = { mode: 'replace' as const }

  const page = useRouteQuery('page', 1, { ...opts, transform: Number })

  const type = mappedAxis('type', LEGACY_RESOURCE_TYPE, RESOURCE_TYPE_OPTIONS)
  const language = mappedAxis(
    'language',
    LEGACY_RESOURCE_LANGUAGE,
    LANGUAGE_OPTIONS
  )
  const platform = mappedAxis(
    'platform',
    LEGACY_RESOURCE_PLATFORM,
    PLATFORM_OPTIONS
  )
  const gameTypeRaw = useRouteQuery<string>('gameType', '', opts)
  const gameType = computed({
    get: () => (gameTypeRaw.value === 'all' ? '' : gameTypeRaw.value),
    set: (value: string) => {
      gameTypeRaw.value = value
    }
  })
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

export const useBrowseWorksQuery = () => {
  const filters = useGalgameFilters()
  const { allowsNsfw } = useContentStance()

  const query = computed<ListWorksQuery>(() => {
    const type = axisKey<NonNullable<ListWorksQuery['resource_type']>>(
      filters.type.value,
      LEGACY_RESOURCE_TYPE,
      RESOURCE_TYPE_OPTIONS
    )
    const platform = axisKey<
      NonNullable<ListWorksQuery['resource_platforms']>[number]
    >(
      filters.platform.value,
      LEGACY_RESOURCE_PLATFORM,
      PLATFORM_OPTIONS
    )
    const language = axisKey<
      NonNullable<ListWorksQuery['resource_languages']>[number]
    >(
      filters.language.value,
      LEGACY_RESOURCE_LANGUAGE,
      LANGUAGE_OPTIONS
    )
    const gameType = GAME_TYPES.has(
      filters.gameType.value as NonNullable<ListWorksQuery['game_type']>
    )
      ? (filters.gameType.value as NonNullable<ListWorksQuery['game_type']>)
      : undefined
    const releasedMonths = csvInts(filters.releasedMonths.value, 1, 12)
    const collectedMonths = csvInts(filters.collectedMonths.value, 1, 12)
    const resourceProviders = csvProviders(filters.includeProviders.value)
    const excludedSoleProviders = csvProviders(
      filters.excludeOnlyProviders.value
    )
    return {
      page: filters.page.value,
      limit: filters.limit,
      sort: browseSortToken(filters.sortField.value, filters.sortOrder.value),
      ...(type ? { resource_type: type } : {}),
      ...(platform ? { resource_platforms: [platform] } : {}),
      ...(language ? { resource_languages: [language] } : {}),
      ...(gameType ? { game_type: gameType } : {}),
      ...(resourceProviders
        ? { resource_providers: resourceProviders }
        : {}),
      ...(excludedSoleProviders
        ? { excluded_sole_providers: excludedSoleProviders }
        : {}),
      ...(filters.releasedFrom.value
        ? { released_from: filters.releasedFrom.value }
        : {}),
      ...(filters.releasedTo.value
        ? { released_to: filters.releasedTo.value }
        : {}),
      ...(releasedMonths ? { released_months: releasedMonths } : {}),
      ...(filters.collectedFrom.value
        ? { collected_from: filters.collectedFrom.value }
        : {}),
      ...(filters.collectedTo.value
        ? { collected_to: filters.collectedTo.value }
        : {}),
      ...(collectedMonths ? { collected_months: collectedMonths } : {}),
      ...(filters.minRating.value > 0
        ? { min_rating: filters.minRating.value }
        : {}),
      ...(filters.minRatingCount.value > 0
        ? { min_rating_count: filters.minRatingCount.value }
        : {}),
      include_nsfw: allowsNsfw.value
    }
  })

  return { ...filters, query }
}
