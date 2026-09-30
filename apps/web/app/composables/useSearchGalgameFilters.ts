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

export const useSearchGalgameFilters = () => {
  const opts = { mode: 'replace' as const }

  const companyId = useRouteQuery('company_id', 0, {
    ...opts,
    transform: Number
  })
  const tagIds = useRouteQuery<string>('tag_ids', '', opts)
  const releasedFrom = useRouteQuery<string>('released_from', '', opts)
  const releasedTo = useRouteQuery<string>('released_to', '', opts)
  const sort = useRouteQuery<string>('sort', 'relevance', opts)

  const tagIdList = computed<number[]>({
    get: () =>
      tagIds.value
        .split(',')
        .map(Number)
        .filter((id) => Number.isInteger(id) && id > 0),
    set: (ids) => {
      tagIds.value = ids.join(',')
    }
  })

  const clear = () => {
    companyId.value = 0
    tagIds.value = ''
    releasedFrom.value = ''
    releasedTo.value = ''
  }

  return {
    companyId,
    tagIds: tagIdList,
    releasedFrom,
    releasedTo,
    sort,
    clear
  }
}
