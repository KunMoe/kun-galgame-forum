import type { ApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import type {
  CharacterSummary,
  CompanySummary,
  CreditNameRef,
  Engine,
  SeriesSummary,
  TagSummary
} from '#shared/utils/api/schemas'
import {
  catalogEntityName,
  catalogVocabularyName,
  type CatalogName
} from '#shared/utils/catalogName'

export const ENTITY_Q_MAX = 100
export const ENTITY_LIMIT_ALL = 8
export const ENTITY_LIMIT_ONE = 24

export const SEARCH_ENTITY_FAMILY_VALUES: SearchEntityFamily[] = [
  'character',
  'company',
  'staff',
  'tag',
  'series',
  'engine'
]

export const trimEntityQ = (q: string) => q.trim().slice(0, ENTITY_Q_MAX)

const displayName = (n: CatalogName, preferOriginal: boolean) =>
  catalogEntityName(n, preferOriginal).name

const ofItem = (
  family: SearchEntityFamily,
  id: string,
  name: string,
  extra?: Partial<SearchEntityItem>
): SearchEntityItem => ({
  id: Number(id),
  family,
  name,
  ...extra
})

const tagItem = (row: TagSummary): SearchEntityItem =>
  ofItem('tag', row.id, catalogVocabularyName(row), {
    work_count: row.catalog_work_count
  })

const companyItem = (
  row: CompanySummary,
  preferOriginal: boolean
): SearchEntityItem => {
  const name = displayName(row, preferOriginal)
  return ofItem('company', row.id, name, {
    alias: row.aliases.find((alias) => alias && alias !== name),
    image: row.logo?.url,
    work_count: row.catalog_work_count
  })
}

const characterItem = (
  row: CharacterSummary,
  preferOriginal: boolean
): SearchEntityItem =>
  ofItem('character', row.id, displayName(row, preferOriginal), {
    image: row.image?.url,
    work_count: row.catalog_work_count
  })

const staffItem = (
  row: CreditNameRef,
  preferOriginal: boolean
): SearchEntityItem =>
  ofItem('staff', row.id, displayName(row, preferOriginal))

const seriesItem = (
  row: SeriesSummary,
  preferOriginal: boolean
): SearchEntityItem =>
  ofItem('series', row.id, displayName(row, preferOriginal), {
    work_count: row.catalog_work_count
  })

const engineItem = (
  row: Engine,
  preferOriginal: boolean
): SearchEntityItem => {
  const name = displayName(row, preferOriginal)
  return ofItem('engine', row.id, name, {
    alias: row.aliases.find((alias) => alias && alias !== name),
    work_count: row.catalog_work_count
  })
}

const failedGroup = (family: SearchEntityFamily): SearchEntityGroup => ({
  family,
  total: 0,
  items: [],
  failed: true
})

export const fetchEntityFamily = async (
  api: ApiClient,
  family: SearchEntityFamily,
  rawQ: string,
  page: number,
  limit: number,
  includeNsfw: boolean,
  preferOriginal: boolean,
  report = true
): Promise<SearchEntityGroup> => {
  const q = trimEntityQ(rawQ)
  if (!q) {
    return { family, total: 0, items: [] }
  }
  const paged = { q, page, limit }
  const shown = { ...paged, include_nsfw: includeNsfw }

  switch (family) {
    case 'tag': {
      const result = await settle(
        api.GET('/tags', { params: { query: shown } })
      )
      if (!result.ok) {
        if (report) reportProblem(result.problem)
        return failedGroup(family)
      }
      return {
        family,
        total: result.data.total,
        items: result.data.items.map(tagItem)
      }
    }
    case 'company': {
      const result = await settle(
        api.GET('/companies', { params: { query: paged } })
      )
      if (!result.ok) {
        if (report) reportProblem(result.problem)
        return failedGroup(family)
      }
      return {
        family,
        total: result.data.total,
        items: result.data.items.map((row) =>
          companyItem(row, preferOriginal)
        )
      }
    }
    case 'character': {
      const result = await settle(
        api.GET('/characters', { params: { query: paged } })
      )
      if (!result.ok) {
        if (report) reportProblem(result.problem)
        return failedGroup(family)
      }
      return {
        family,
        total: result.data.total,
        items: result.data.items.map((row) =>
          characterItem(row, preferOriginal)
        )
      }
    }
    case 'staff': {
      const result = await settle(
        api.GET('/credit-names', { params: { query: paged } })
      )
      if (!result.ok) {
        if (report) reportProblem(result.problem)
        return failedGroup(family)
      }
      return {
        family,
        total: result.data.total,
        items: result.data.items.map((row) => staffItem(row, preferOriginal))
      }
    }
    case 'series': {
      const result = await settle(
        api.GET('/series', { params: { query: shown } })
      )
      if (!result.ok) {
        if (report) reportProblem(result.problem)
        return failedGroup(family)
      }
      return {
        family,
        total: result.data.total,
        items: result.data.items.map((row) =>
          seriesItem(row, preferOriginal)
        )
      }
    }
    case 'engine': {
      const result = await settle(
        api.GET('/engines', { params: { query: paged } })
      )
      if (!result.ok) {
        if (report) reportProblem(result.problem)
        return failedGroup(family)
      }
      return {
        family,
        total: result.data.total,
        items: result.data.items.map((row) =>
          engineItem(row, preferOriginal)
        )
      }
    }
  }
}

export const fetchEntityFamilies = (
  api: ApiClient,
  q: string,
  page: number,
  limit: number,
  includeNsfw: boolean,
  preferOriginal: boolean,
  families: SearchEntityFamily[],
  report = true
) =>
  Promise.all(
    families.map((family) =>
      fetchEntityFamily(
        api,
        family,
        q,
        page,
        limit,
        includeNsfw,
        preferOriginal,
        report
      )
    )
  )
