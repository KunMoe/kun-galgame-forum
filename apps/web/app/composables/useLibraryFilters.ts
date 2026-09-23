import { useRouteQuery } from '@vueuse/router'
import type { ListLibraryWorksQuery } from '#shared/utils/api/schemas'

type LibrarySortField = 'popularity' | 'release_date' | 'time' | 'relevance'

export const librarySortToken = (
  field: string,
  order: string
): NonNullable<ListLibraryWorksQuery['sort']> => {
  if (
    field === 'release_date' ||
    field === 'released_desc' ||
    field === 'released_asc'
  ) {
    return order === 'asc' || field === 'released_asc'
      ? 'released_asc'
      : 'released_desc'
  }
  if (field === 'time' || field === 'updated' || field === 'updated_desc') {
    return 'updated_desc'
  }
  if (field === 'relevance' || field === 'relevance_desc') {
    return 'relevance_desc'
  }
  return 'popularity_desc'
}

// popularity is a catalog sort and the local list cannot answer it, so only the
// library page may ask for it as its default.
export const useLibraryFilters = () => {
  const opts = { mode: 'replace' as const }

  const page = useRouteQuery('page', 1, { ...opts, transform: Number })
  const sortField = useRouteQuery<LibrarySortField>(
    'sortField',
    'popularity',
    opts
  )
  const sortOrder = useRouteQuery<KunOrder>('sortOrder', 'desc', opts)
  const releasedFrom = useRouteQuery<string>('releasedFrom', '', opts)
  const releasedTo = useRouteQuery<string>('releasedTo', '', opts)
  const limit = 24

  return {
    page,
    limit,
    sortField,
    sortOrder,
    releasedFrom,
    releasedTo
  }
}

export const useLibraryWorksQuery = () => {
  const filters = useLibraryFilters()
  const { allowsNsfw } = useContentStance()

  const query = computed<ListLibraryWorksQuery>(() => ({
    page: filters.page.value,
    limit: filters.limit,
    sort: librarySortToken(filters.sortField.value, filters.sortOrder.value),
    ...(filters.releasedFrom.value
      ? { released_from: filters.releasedFrom.value }
      : {}),
    ...(filters.releasedTo.value
      ? { released_to: filters.releasedTo.value }
      : {}),
    include_nsfw: allowsNsfw.value
  }))

  return { ...filters, query }
}
