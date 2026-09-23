import type {
  Company,
  CompanyWorkPage,
  WorksQuery
} from '#shared/utils/api/schemas'
import { mergedInto } from '#shared/utils/api/merged'
import { workSummaryToCard } from '~/utils/galgame/workCard'

export const useGalgameOfficialDetail = async (subPath = '') => {
  const route = useRoute()
  const officialId = Number((route.params as { id: string }).id)

  if (!Number.isInteger(officialId) || officialId <= 0) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到 Galgame 会社',
      fatal: true
    })
  }

  const nuxtApp = useNuxtApp()
  const nameOf = useCatalogName()

  let movedTo: number | null = null
  const { data: company } = await useApi<Company>(
    () => `company:${officialId}`,
    async (api) => {
      const res = await api.GET('/companies/{company_id}', {
        params: { path: { company_id: String(officialId) } }
      })
      movedTo = mergedInto(res.error)
      return res
    }
  )

  if (movedTo) {
    const target = `${taxonomyDetailPath('official', movedTo)}${subPath}`
    await nuxtApp.runWithContext(() =>
      navigateTo(target, { redirectCode: 301, replace: true })
    )
    return { officialId, data: computed(() => null) }
  }

  if (!company.value) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到 Galgame 会社',
      fatal: true
    })
  }

  const data = computed(() => {
    const c = company.value
    if (!c) {
      return null
    }
    const { name, original } = nameOf(c)
    const intro = pickCatalogIntro(c.intros)
    return {
      id: Number(c.id),
      name,
      original,
      links: c.links.map((l) => ({
        url: l.url,
        name: catalogLinkLabel(l.site, l.url)
      })),
      logo: c.logo?.url ?? '',
      category: c.company_kind,
      lang: c.lang ?? '',
      description: intro?.value ?? '',
      description_machine: intro?.is_machine ?? false,
      alias: c.aliases
    }
  })

  return { officialId, data }
}

export const useCompanyWorks = async (
  officialId: number,
  query: () => WorksQuery
) => {
  const nameOf = useCatalogName()
  // Both reads start before either is awaited: a composable resumed after an
  // await has lost the Nuxt instance, so a second useApi there throws on SSR.
  const pageRead = useApi<CompanyWorkPage>(
    () => `company-works:${officialId}:${JSON.stringify(query())}`,
    (client) =>
      client.GET('/companies/{company_id}/works', {
        params: { path: { company_id: String(officialId) }, query: query() }
      })
  )
  // Own and imprint are the same collection split by via=; a one-row page of
  // each carries its total.
  const splitRead = useApi<{ own: number; imprint: number }>(
    () =>
      `company-works-split:${officialId}:${JSON.stringify({ ...query(), page: 1 })}`,
    async (client) => {
      const [own, imprint] = await Promise.all(
        (['own', 'imprint'] as const).map((via) =>
          client.GET('/companies/{company_id}/works', {
            params: {
              path: { company_id: String(officialId) },
              query: { ...query(), via, page: 1, limit: 1 }
            }
          })
        )
      )
      return {
        data: { own: own?.data?.total ?? 0, imprint: imprint?.data?.total ?? 0 },
        response: new Response(null, { status: 200 })
      }
    }
  )
  const [{ data: page, status }, { data: split }] = await Promise.all([
    pageRead,
    splitRead
  ])

  const galgames = computed(() =>
    (page.value?.items ?? []).map((item) => ({
      ...workSummaryToCard(item.work_summary, nameOf),
      via_official: item.via_company
        ? {
            id: Number(item.via_company.id),
            name: nameOf(item.via_company).name
          }
        : undefined
    }))
  )
  const total = computed(() => page.value?.total ?? 0)
  const own = computed(() => split.value?.own ?? 0)
  const imprint = computed(() => split.value?.imprint ?? 0)

  return { galgames, total, status, own, imprint }
}
