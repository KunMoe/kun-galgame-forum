import { settle } from '#shared/utils/api/problem'
import type { CompanySummary, TagSummary } from '#shared/utils/api/schemas'
import { catalogVocabularyName } from '#shared/utils/catalogName'

type EntityNameFamily = 'company' | 'tag'

const IDS_PER_CALL = 100

// A filtered list keeps its 会社 and 标签 in the URL as ids so it can be shared,
// and a link opened cold arrives with nothing to draw its chips with until
// catalog answers the names.
export const useEntityNames = () => {
  const named = reactive(new Map<string, SearchEntityItem>())
  const api = useApiClient()
  const nameOf = useCatalogName()
  const { allowsNsfw } = useContentStance()

  const keyOf = (family: string, id: number) => `${family}:${id}`

  const remember = (item: SearchEntityItem) =>
    named.set(keyOf(item.family, item.id), item)

  const itemOf = (family: EntityNameFamily, id: number) =>
    named.get(keyOf(family, id))

  const labelOf = (family: EntityNameFamily, id: number) =>
    itemOf(family, id)?.name ?? `#${id}`

  const itemsOf = (family: EntityNameFamily, ids: number[]) =>
    ids
      .map((id) => itemOf(family, id))
      .filter((item): item is SearchEntityItem => !!item)

  const rememberTag = (row: TagSummary) =>
    remember({
      id: Number(row.id),
      family: 'tag',
      name: catalogVocabularyName(row),
      work_count: row.catalog_work_count
    })

  const rememberCompany = (row: CompanySummary) => {
    const { name } = nameOf(row)
    remember({
      id: Number(row.id),
      family: 'company',
      name,
      alias: row.aliases.find((alias) => alias && alias !== name),
      image: row.logo?.url,
      work_count: row.catalog_work_count
    })
  }

  const resolve = async (wanted: Partial<Record<EntityNameFamily, number[]>>) =>
    Promise.all(
      (Object.keys(wanted) as EntityNameFamily[]).map(async (family) => {
        const missing = (wanted[family] ?? []).filter(
          (id) => id > 0 && !named.has(keyOf(family, id))
        )
        if (!missing.length) {
          return
        }
        for (let i = 0; i < missing.length; i += IDS_PER_CALL) {
          const ids = missing.slice(i, i + IDS_PER_CALL).map(String)
          const result =
            family === 'tag'
              ? await settle(
                  api.GET('/tags', {
                    params: {
                      query: {
                        ids,
                        limit: ids.length,
                        include_nsfw: allowsNsfw.value
                      }
                    }
                  })
                )
              : await settle(
                  api.GET('/companies', {
                    params: { query: { ids, limit: ids.length } }
                  })
                )
          if (!result.ok) {
            reportProblem(result.problem)
            continue
          }
          if (family === 'tag') {
            for (const row of result.data.items) {
              rememberTag(row as TagSummary)
            }
          } else {
            for (const row of result.data.items) {
              rememberCompany(row as CompanySummary)
            }
          }
        }
      })
    )

  return { remember, labelOf, itemsOf, resolve }
}
