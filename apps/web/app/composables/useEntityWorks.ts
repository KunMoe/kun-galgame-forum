import type { WorksQuery, WorkSummary } from '#shared/utils/api/schemas'
import { emulatorRuntimeFilter } from '#shared/utils/galgameResourceVocab'
import { workSummaryToCard } from '~/utils/galgame/workCard'
import { browseSortToken } from './useGalgameFilters'

export const useEntityWorksQuery = () => {
  const { page, limit, state } = useGalgameFilters()
  const { allowsNsfw } = useContentStance()

  const query = computed<WorksQuery>(() => {
    const f = state.value
    return {
      page: f.page,
      limit,
      sort: browseSortToken(f.sortField, f.sortOrder) as WorksQuery['sort'],
      resource_type: (f.type || undefined) as WorksQuery['resource_type'],
      resource_platform: f.platform.find(
        (platform) => platform !== 'emulator'
      ) as WorksQuery['resource_platform'],
      resource_runtime: emulatorRuntimeFilter(f.platform, f.runtime)?.[0],
      resource_language: f.language[0] as WorksQuery['resource_language'],
      game_type: f.gameType || undefined,
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
