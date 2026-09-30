import type { ListWorksQuery } from '#shared/utils/api/schemas'
import {
  EMULATOR_RUNTIME_OPTIONS,
  LANGUAGE_OPTIONS,
  LEGACY_RESOURCE_LANGUAGE,
  LEGACY_RESOURCE_PLATFORM,
  LEGACY_RESOURCE_TYPE,
  PLATFORM_FILTER_OPTIONS,
  RESOURCE_TYPE_OPTIONS,
  axisKey,
  emulatorRuntimeFilter
} from '#shared/utils/galgameResourceVocab'
import {
  PROVIDER_KEY_OPTIONS,
  type ProviderKey
} from '~/constants/galgameResource'

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

const GAME_TYPES = [
  '',
  'ba_saku',
  'plot',
  'moe',
  'daily',
  'uncategorized'
] as const

const inVocabOrder = (
  values: string[],
  legacy: Record<string, string>,
  options: { value: string }[]
) => {
  const keys = new Set(values.map((value) => axisKey(value, legacy, options)))
  return options.map((option) => option.value).filter((key) => keys.has(key))
}

const axisField = (
  legacy: Record<string, string>,
  options: { value: string }[]
): QueryField<string> => ({
  parse: (raw) => axisKey(raw ?? '', legacy, options) ?? '',
  format: (value) => value
})

const axisListField = (
  legacy: Record<string, string>,
  options: { value: string }[]
): QueryField<string[]> => ({
  parse: (raw) => inVocabOrder((raw ?? '').split(','), legacy, options),
  format: (values) => inVocabOrder(values, legacy, options).join(',')
})

const monthListField: QueryField<number[]> = {
  parse: (raw) =>
    [...new Set((raw ?? '').split(',').map(Number))]
      .filter((month) => Number.isInteger(month) && month >= 1 && month <= 12)
      .sort((a, b) => a - b),
  format: (months) => [...months].sort((a, b) => a - b).join(',')
}

const providerListField: QueryField<ProviderKey[]> = {
  parse: (raw) => {
    const picked = new Set((raw ?? '').split(','))
    return PROVIDER_KEY_OPTIONS.filter((key) => picked.has(key))
  },
  format: (keys) =>
    PROVIDER_KEY_OPTIONS.filter((key) => keys.includes(key)).join(',')
}

export const GALGAME_FILTER_SCHEMA = {
  page: queryPage(),
  type: axisField(LEGACY_RESOURCE_TYPE, RESOURCE_TYPE_OPTIONS),
  language: axisListField(LEGACY_RESOURCE_LANGUAGE, LANGUAGE_OPTIONS),
  platform: axisListField(LEGACY_RESOURCE_PLATFORM, PLATFORM_FILTER_OPTIONS),
  runtime: axisListField({}, EMULATOR_RUNTIME_OPTIONS),
  gameType: queryEnum(GAME_TYPES, ''),
  sortField: queryString('time'),
  sortOrder: queryEnum(['desc', 'asc'] as const, 'desc'),
  releasedFrom: queryString(),
  releasedTo: queryString(),
  releasedMonths: monthListField,
  collectedFrom: queryString(),
  collectedTo: queryString(),
  collectedMonths: monthListField,
  includeProviders: providerListField,
  excludeOnlyProviders: providerListField,
  minRatingCount: queryInt(0, 0),
  minRating: queryInt(0, 0)
}

export const GALGAME_FILTER_QUERY_KEYS = Object.keys(GALGAME_FILTER_SCHEMA)

export const browseSortToken = (
  field: string,
  order: string
): NonNullable<ListWorksQuery['sort']> => {
  const key =
    field === 'time' ? 'resource_updated' : field === 'views' ? 'view' : field
  const dir = order === 'asc' ? 'asc' : 'desc'
  const token = `${key}_${dir}`
  return BROWSE_SORT_FIELDS.has(key)
    ? (token as NonNullable<ListWorksQuery['sort']>)
    : 'resource_updated_desc'
}

export const useGalgameFilters = () => ({
  ...useQueryState(GALGAME_FILTER_SCHEMA, { pageKey: 'page' }),
  limit: 24
})

export const useBrowseWorksQuery = () => {
  const filters = useGalgameFilters()
  const { allowsNsfw } = useContentStance()

  const query = computed<ListWorksQuery>(() => {
    const f = filters.state.value
    const platforms = f.platform.filter(
      (platform) => platform !== 'emulator'
    ) as NonNullable<ListWorksQuery['resource_platforms']>
    const runtimes = emulatorRuntimeFilter(f.platform, f.runtime)
    const languages = f.language as NonNullable<
      ListWorksQuery['resource_languages']
    >
    return {
      page: f.page,
      limit: filters.limit,
      sort: browseSortToken(f.sortField, f.sortOrder),
      ...(f.type
        ? { resource_type: f.type as ListWorksQuery['resource_type'] }
        : {}),
      ...(platforms.length ? { resource_platforms: platforms } : {}),
      ...(runtimes ? { resource_runtimes: runtimes } : {}),
      ...(languages.length ? { resource_languages: languages } : {}),
      ...(f.gameType ? { game_type: f.gameType } : {}),
      ...(f.includeProviders.length
        ? { resource_providers: f.includeProviders }
        : {}),
      ...(f.excludeOnlyProviders.length
        ? { excluded_sole_providers: f.excludeOnlyProviders }
        : {}),
      ...(f.releasedFrom ? { released_from: f.releasedFrom } : {}),
      ...(f.releasedTo ? { released_to: f.releasedTo } : {}),
      ...(f.releasedMonths.length ? { released_months: f.releasedMonths } : {}),
      ...(f.collectedFrom ? { collected_from: f.collectedFrom } : {}),
      ...(f.collectedTo ? { collected_to: f.collectedTo } : {}),
      ...(f.collectedMonths.length
        ? { collected_months: f.collectedMonths }
        : {}),
      ...(f.minRating > 0 ? { min_rating: f.minRating } : {}),
      ...(f.minRatingCount > 0 ? { min_rating_count: f.minRatingCount } : {}),
      include_nsfw: allowsNsfw.value
    }
  })

  return { ...filters, query }
}
