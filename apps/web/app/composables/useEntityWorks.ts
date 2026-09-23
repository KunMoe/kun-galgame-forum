import type { WorksQuery, WorkSummary } from '#shared/utils/api/schemas'
import {
  LANGUAGE_OPTIONS,
  LEGACY_RESOURCE_LANGUAGE,
  LEGACY_RESOURCE_PLATFORM,
  LEGACY_RESOURCE_TYPE,
  PLATFORM_OPTIONS,
  RESOURCE_TYPE_OPTIONS,
  axisKey
} from '#shared/utils/galgameResourceVocab'
import { workSummaryToCard } from '~/utils/galgame/workCard'
import { browseSortToken } from './useGalgameFilters'

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
    const sort = browseSortToken(
      sortField.value,
      sortOrder.value
    ) as WorksQuery['sort']
    return {
      page: page.value,
      limit,
      sort,
      resource_type: axisKey<NonNullable<WorksQuery['resource_type']>>(
        type.value,
        LEGACY_RESOURCE_TYPE,
        RESOURCE_TYPE_OPTIONS
      ),
      resource_platform: axisKey<
        NonNullable<WorksQuery['resource_platform']>
      >(
        platform.value,
        LEGACY_RESOURCE_PLATFORM,
        PLATFORM_OPTIONS
      ),
      resource_language: axisKey<
        NonNullable<WorksQuery['resource_language']>
      >(
        language.value,
        LEGACY_RESOURCE_LANGUAGE,
        LANGUAGE_OPTIONS
      ),
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
