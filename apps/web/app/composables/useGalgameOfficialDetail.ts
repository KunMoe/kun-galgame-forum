import { isMissingProblem } from '#shared/utils/api/problem'
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

  const nameOf = useCatalogName()
  const notFound = '未找到 Galgame 会社'

  const movedTo = ref<number | null>(null)
  const read = useKunApi<Company>(
    () => `company:${officialId}`,
    async (api, { signal }) => {
      const res = await api.GET('/companies/{company_id}', {
        params: { path: { company_id: String(officialId) } },
        signal
      })
      movedTo.value = mergedInto(res.error)
      return res
    },
    { critical: true }
  )
  const company = read.data
  const redirect = (id: number) =>
    navigateTo(`${taxonomyDetailPath('official', id)}${subPath}`, {
      redirectCode: 301,
      replace: true
    })

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
      alias: c.aliases,
      follower_count: c.follower_count
    }
  })

  const detail = { officialId, data, pending: read.pending }

  // Watchers go on before the await: past it this composable has lost its
  // component instance.
  if (import.meta.client) {
    watch(movedTo, (id) => {
      if (id) {
        redirect(id)
      }
    })
    useKunNotFound(
      () => isMissingProblem(read.problem.value) && movedTo.value === null,
      notFound
    )
    return detail
  }

  const nuxtApp = useNuxtApp()
  await read
  if (movedTo.value) {
    const id = movedTo.value
    await nuxtApp.runWithContext(() => redirect(id))
    return detail
  }
  useKunNotFound(() => isMissingProblem(read.problem.value), notFound)
  return detail
}

export const useCompanyWorks = async (
  officialId: number,
  baseQuery: () => WorksQuery
) => {
  const nameOf = useCatalogName()
  const { excludeQuery } = useTagBlocks()
  const query = (): WorksQuery => ({ ...baseQuery(), ...excludeQuery.value })
  // Both reads start before either is awaited: a composable resumed after an
  // await has lost the Nuxt instance, so a second useKunApi there throws on SSR.
  const pageRead = useKunApi<CompanyWorkPage>(
    () => `company-works:${officialId}:${JSON.stringify(query())}`,
    (client) =>
      client.GET('/companies/{company_id}/works', {
        params: { path: { company_id: String(officialId) }, query: query() }
      })
  )
  // Own and imprint are the same collection split by via=; a one-row page of
  // each carries its total.
  const splitRead = useKunApi<{ own: number; imprint: number }>(
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
        ok: true as const,
        data: { own: own?.data?.total ?? 0, imprint: imprint?.data?.total ?? 0 }
      }
    }
  )
  const galgames = computed(() =>
    (pageRead.data.value?.items ?? []).map((item) => ({
      ...workSummaryToCard(item.work_summary, nameOf),
      via_official: item.via_company
        ? {
            id: Number(item.via_company.id),
            name: nameOf(item.via_company).name
          }
        : undefined
    }))
  )
  const total = computed(() => pageRead.data.value?.total ?? 0)
  const own = computed(() => splitRead.data.value?.own ?? 0)
  const imprint = computed(() => splitRead.data.value?.imprint ?? 0)

  await Promise.all([pageRead, splitRead])

  return {
    galgames,
    total,
    status: pageRead.status,
    pending: pageRead.pending,
    own,
    imprint
  }
}
