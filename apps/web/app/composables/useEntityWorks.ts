import type { WorksQuery, WorkSummary } from '#shared/utils/api/schemas'
import {
  LANGUAGE_OPTIONS,
  PLATFORM_OPTIONS,
  RESOURCE_TYPE_OPTIONS
} from '#shared/utils/galgameResourceVocab'
import { workSummaryToCard } from '~/utils/galgame/workCard'

// Links shared before the entity pages moved to v1 still carry the legacy
// resource scalars; they keep working as the axis key they meant.
const legacyPlatform: Record<string, string> = {
  windows: 'win',
  app: 'and',
  linux: 'lin',
  others: 'oth'
}
const legacyLanguage: Record<string, string> = { others: 'other' }
const legacyType: Record<string, string> = { others: 'other', image: 'cg' }

const axisKey = <T extends string>(
  value: string,
  legacy: Record<string, string>,
  options: { value: string }[]
): T | undefined => {
  if (!value || value === 'all') {
    return undefined
  }
  const key = legacy[value] ?? value
  return options.some((o) => o.value === key) ? (key as T) : undefined
}

const sortTokens = new Set([
  'resource_updated',
  'created',
  'view',
  'view_1d',
  'view_7d',
  'view_30d',
  'release_date',
  'rating'
])

export const useEntityWorksQuery = () => {
  const {
    page,
    limit,
    type,
    language,
    platform,
    gameType,
    sortField,
    sortOrder
  } = useGalgameFilters()
  const { allowsNsfw } = useContentStance()

  const query = computed<WorksQuery>(() => {
    const field = sortField.value === 'time' ? 'resource_updated' : sortField.value
    const order = sortOrder.value === 'asc' ? 'asc' : 'desc'
    const sort = (
      sortTokens.has(field) ? `${field}_${order}` : 'resource_updated_desc'
    ) as WorksQuery['sort']
    return {
      page: page.value,
      limit,
      sort,
      resource_type: axisKey(type.value, legacyType, RESOURCE_TYPE_OPTIONS),
      resource_platform: axisKey(platform.value, legacyPlatform, PLATFORM_OPTIONS),
      resource_language: axisKey(language.value, legacyLanguage, LANGUAGE_OPTIONS),
      game_type:
        gameType.value && gameType.value !== 'all'
          ? (gameType.value as WorksQuery['game_type'])
          : undefined,
      include_nsfw: allowsNsfw.value
    }
  })

  return { page, limit, query }
}

export const useWorkCards = (works: () => WorkSummary[] | undefined) => {
  const nameOf = useCatalogName()
  return computed(() => (works() ?? []).map((w) => workSummaryToCard(w, nameOf)))
}
