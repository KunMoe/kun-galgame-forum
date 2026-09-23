import { useRouteQuery } from '@vueuse/router'
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

export const useToolsetFilters = () => {
  const opts = { mode: 'replace' as const }

  const page = useRouteQuery('page', 1, { ...opts, transform: Number })
  const type = useRouteQuery<ToolsetTypeFilter>('type', 'all', opts)
  const language = useRouteQuery<ToolsetLanguageFilter>('language', 'all', opts)
  const platform = useRouteQuery<ToolsetPlatformFilter>('platform', 'all', opts)
  const version = useRouteQuery<ToolsetVersionFilter>('version', 'all', opts)
  const sortField = useRouteQuery<ToolsetSortField>(
    'sortField',
    'resource_update_time',
    opts
  )
  const sortOrder = useRouteQuery<KunOrder>('sortOrder', 'desc', opts)

  const query = useRouteQuery<string>('query', '', opts)

  const limit = 24

  const sort = computed<ToolsetSort>(() => {
    const order = sortOrder.value === 'asc' ? 'asc' : 'desc'
    if (sortField.value === 'created') {
      return `created_${order}`
    }
    if (sortField.value === 'view') {
      return `view_${order}`
    }
    return `resource_updated_${order}`
  })

  const listQuery = computed(() => {
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
      page: page.value,
      limit,
      sort: sort.value
    }
    if (type.value !== 'all') {
      params.toolset_type = type.value
    }
    if (language.value !== 'all') {
      params.interface_language = language.value
    }
    if (platform.value !== 'all') {
      params.platform = platform.value
    }
    if (version.value !== 'all') {
      params.release_channel = version.value
    }
    const q = query.value.trim()
    if (q) {
      params.q = q
    }
    return params
  })

  return {
    page,
    limit,
    type,
    language,
    platform,
    version,
    sortField,
    sortOrder,
    query,
    listQuery
  }
}
