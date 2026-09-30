// The Galgame lane's filters live in the URL so a filtered search can be
// shared. They belong to that one lane, so SearchContainer drops these keys
// when the category changes — the list is here rather than there because the
// two have to agree.
export const SEARCH_GALGAME_FILTER_KEYS = [
  'company_id',
  'tag_ids',
  'released_from',
  'released_to',
  'sort'
] as const

const SEARCH_GALGAME_FILTER_SCHEMA = {
  page: queryPage(),
  company_id: queryInt(0, 0),
  tag_ids: queryIds(),
  released_from: queryString(),
  released_to: queryString(),
  sort: queryString('relevance')
}

export const useSearchGalgameFilters = () => {
  const filters = useQueryState(SEARCH_GALGAME_FILTER_SCHEMA, {
    pageKey: 'page'
  })
  const clear = () =>
    filters.set({
      company_id: 0,
      tag_ids: [],
      released_from: '',
      released_to: ''
    })
  return { ...filters, clear }
}
