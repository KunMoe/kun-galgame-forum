import type {
  ToolsetInterfaceLanguage,
  ToolsetPlatform,
  ToolsetReleaseChannel,
  ToolsetSort,
  ToolsetType
} from '#shared/utils/api/schemas'

type ToolsetTypeFilter = 'all' | ToolsetType
type ToolsetLanguageFilter = 'all' | ToolsetInterfaceLanguage
type ToolsetPlatformFilter = 'all' | ToolsetPlatform
type ToolsetVersionFilter = 'all' | ToolsetReleaseChannel
type ToolsetSortField = 'resource_update_time' | 'created' | 'view'

const TOOLSET_FILTER_SCHEMA = {
  page: queryPage(),
  type: queryString('all') as QueryField<ToolsetTypeFilter>,
  language: queryString('all') as QueryField<ToolsetLanguageFilter>,
  platform: queryString('all') as QueryField<ToolsetPlatformFilter>,
  version: queryString('all') as QueryField<ToolsetVersionFilter>,
  sortField: queryString(
    'resource_update_time'
  ) as QueryField<ToolsetSortField>,
  sortOrder: queryEnum(['desc', 'asc'] as const, 'desc'),
  query: queryString()
}

export const useToolsetFilters = () => {
  const filters = useQueryState(TOOLSET_FILTER_SCHEMA, { pageKey: 'page' })
  const limit = 24

  const listQuery = computed(() => {
    const f = filters.state.value
    const params: {
      page: number
      limit: number
      sort: ToolsetSort
      toolset_type?: ToolsetType
      interface_language?: ToolsetInterfaceLanguage
      platform?: ToolsetPlatform
      release_channel?: ToolsetReleaseChannel
      q?: string
    } = {
      page: f.page,
      limit,
      sort:
        f.sortField === 'created' || f.sortField === 'view'
          ? `${f.sortField}_${f.sortOrder}`
          : `resource_updated_${f.sortOrder}`
    }
    if (f.type !== 'all') {
      params.toolset_type = f.type
    }
    if (f.language !== 'all') {
      params.interface_language = f.language
    }
    if (f.platform !== 'all') {
      params.platform = f.platform
    }
    if (f.version !== 'all') {
      params.release_channel = f.version
    }
    const q = f.query.trim()
    if (q) {
      params.q = q
    }
    return params
  })

  return { ...filters, limit, listQuery }
}
