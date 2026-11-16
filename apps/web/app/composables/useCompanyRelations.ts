import type { CompanyGraph } from '#shared/utils/api/schemas'

export const useCompanyRelations = (companyId: number) => {
  const nameOf = useCatalogName()
  const read = useKunApi<CompanyGraph>(
    () => `company-graph:${companyId}`,
    (api, { signal }) =>
      api.GET('/companies/{company_id}/graph', {
        params: { path: { company_id: String(companyId) } },
        signal
      })
  )

  const graph = computed<GalgameOfficialRelationGraph | null>(() => {
    const g = read.data.value
    if (!g || g.nodes.length < 2) {
      return null
    }
    return {
      nodes: g.nodes.map((n) => ({
        id: Number(n.id),
        name: nameOf(n).name,
        logo: n.logo?.url ?? '',
        work_count: n.catalog_work_count
      })),
      edges: g.edges.map((e) => ({
        from: Number(e.from_company_id),
        to: Number(e.to_company_id),
        relation: e.relation
      }))
    }
  })

  const layout = computed(() =>
    graph.value ? buildOfficialGraphLayout(graph.value) : null
  )

  const hasRelations = computed(() => (layout.value?.edges.length ?? 0) > 0)

  const relatedCount = computed(() =>
    hasRelations.value ? (graph.value?.nodes.length ?? 1) - 1 : 0
  )

  return { graph, layout, hasRelations, relatedCount }
}
