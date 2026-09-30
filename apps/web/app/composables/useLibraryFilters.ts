import type { ListLibraryWorksQuery } from '#shared/utils/api/schemas'

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
const LIBRARY_FILTER_SCHEMA = {
  page: queryPage(),
  sortField: queryString('popularity'),
  sortOrder: queryEnum(['desc', 'asc'] as const, 'desc'),
  releasedFrom: queryString(),
  releasedTo: queryString()
}

export const useLibraryFilters = () => ({
  ...useQueryState(LIBRARY_FILTER_SCHEMA, { pageKey: 'page' }),
  limit: 24
})

export const useLibraryWorksQuery = () => {
  const filters = useLibraryFilters()
  const { allowsNsfw } = useContentStance()

  const query = computed<ListLibraryWorksQuery>(() => {
    const f = filters.state.value
    return {
      page: f.page,
      limit: filters.limit,
      sort: librarySortToken(f.sortField, f.sortOrder),
      ...(f.releasedFrom ? { released_from: f.releasedFrom } : {}),
      ...(f.releasedTo ? { released_to: f.releasedTo } : {}),
      include_nsfw: allowsNsfw.value
    }
  })

  return { ...filters, query }
}
