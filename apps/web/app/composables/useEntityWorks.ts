import type { WorksQuery, WorkSummary } from '#shared/utils/api/schemas'
import {
  LEGACY_RESOURCE_TYPE,
  RESOURCE_TYPE_OPTIONS,
  axisKey,
  emulatorRuntimeFilter
} from '#shared/utils/galgameResourceVocab'
import { workSummaryToCard } from '~/utils/galgame/workCard'
import { browseSortToken } from './useGalgameFilters'

export const useEntityWorksQuery = () => {
  const {
    page,
    limit,
    type,
    languages,
    platforms,
    runtimes,
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
      resource_platform: platforms.value.find(
        (platform) => platform !== 'emulator'
      ) as WorksQuery['resource_platform'],
      resource_runtime: emulatorRuntimeFilter(
        platforms.value,
        runtimes.value
      )?.[0],
      resource_language: languages.value[0] as WorksQuery['resource_language'],
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
  return computed(() =>
    (works() ?? []).map((w) => workSummaryToCard(w, nameOf))
  )
}
