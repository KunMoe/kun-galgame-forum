import type { WorkFacets } from '#shared/utils/api/schemas'

export const useEntityFacets = (
  facets: () => WorkFacets | undefined,
  entityNames = useEntityNames()
) => {
  const named = (family: 'tag' | 'company', rows: WorkFacets['tag_counts']) =>
    rows.flatMap((row) => {
      const item = entityNames.itemsOf(family, [Number(row.id)])[0]
      return item ? [{ ...item, work_count: row.work_count }] : []
    })

  const tags = computed(() => named('tag', facets()?.tag_counts ?? []))
  const companies = computed(() =>
    named('company', facets()?.company_counts ?? [])
  )

  watch(
    facets,
    (f) => {
      if (f) {
        void entityNames.resolve({
          tag: f.tag_counts.map((row) => Number(row.id)),
          company: f.company_counts.map((row) => Number(row.id))
        })
      }
    },
    { immediate: true }
  )

  return { tags, companies }
}
