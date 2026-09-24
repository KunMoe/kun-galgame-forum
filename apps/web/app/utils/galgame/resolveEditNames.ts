import type { ApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import {
  catalogVocabularyName,
  type CatalogName
} from '#shared/utils/catalogName'

type NameOf = (n: CatalogName) => { name: string; original: string }

export type EditNameIds = {
  tag: number[]
  official: number[]
  engine: number[]
  series: number[]
  character: number[]
  staff: number[]
}

export type EditNameMaps = { [K in keyof EditNameIds]: Map<number, string> }

const BATCH_SIZE = 100
const CONCURRENCY = 4

const chunks = <T>(items: T[], size: number): T[][] => {
  const out: T[][] = []
  for (let i = 0; i < items.length; i += size) {
    out.push(items.slice(i, i + size))
  }
  return out
}

const eachLimited = async <T>(
  items: T[],
  limit: number,
  run: (item: T) => Promise<void>
) => {
  let next = 0
  const worker = async () => {
    while (next < items.length) {
      const item = items[next++] as T
      await run(item)
    }
  }
  await Promise.all(
    Array.from({ length: Math.min(limit, items.length) }, worker)
  )
}

const unknownIds = (map: Map<number, string>, ids: number[]) => [
  ...new Set(ids.filter((id) => !map.has(id)))
]

// A name that cannot be read (the entity is gone, merged away, or the service
// is down) stays out of the map, and the field shows the raw id.
export const resolveEditNames = async (
  api: ApiClient,
  ids: EditNameIds,
  nameOf: NameOf,
  maps: EditNameMaps
): Promise<void> => {
  const label = (n: CatalogName) => nameOf(n).name

  const tags = chunks(unknownIds(maps.tag, ids.tag), BATCH_SIZE).map(
    async (batch) => {
      const res = await settle(
        api.GET('/tags', {
          params: {
            query: {
              ids: batch.map(String),
              include_nsfw: true,
              limit: BATCH_SIZE
            }
          }
        })
      )
      for (const item of res.ok ? res.data.items : []) {
        maps.tag.set(Number(item.id), catalogVocabularyName(item))
      }
    }
  )

  const companies = chunks(
    unknownIds(maps.official, ids.official),
    BATCH_SIZE
  ).map(async (batch) => {
    const res = await settle(
      api.GET('/companies', {
        params: { query: { ids: batch.map(String), limit: BATCH_SIZE } }
      })
    )
    for (const item of res.ok ? res.data.items : []) {
      maps.official.set(Number(item.id), label(item))
    }
  })

  const singles: Array<{
    map: Map<number, string>
    read: (id: string) => Promise<CatalogName | undefined>
    id: number
  }> = [
    ...unknownIds(maps.engine, ids.engine).map((id) => ({
      map: maps.engine,
      id,
      read: async (engineId: string) => {
        const res = await settle(
          api.GET('/engines/{engine_id}', {
            params: { path: { engine_id: engineId } }
          })
        )
        return res.ok ? res.data : undefined
      }
    })),
    ...unknownIds(maps.series, ids.series).map((id) => ({
      map: maps.series,
      id,
      read: async (seriesId: string) => {
        const res = await settle(
          api.GET('/series/{series_id}', {
            params: { path: { series_id: seriesId } }
          })
        )
        return res.ok ? res.data : undefined
      }
    })),
    ...unknownIds(maps.character, ids.character).map((id) => ({
      map: maps.character,
      id,
      read: async (characterId: string) => {
        const res = await settle(
          api.GET('/characters/{character_id}', {
            params: {
              path: { character_id: characterId },
              query: { include_nsfw: true }
            }
          })
        )
        return res.ok ? res.data : undefined
      }
    })),
    ...unknownIds(maps.staff, ids.staff).map((id) => ({
      map: maps.staff,
      id,
      read: async (creditNameId: string) => {
        const res = await settle(
          api.GET('/credit-names/{credit_name_id}', {
            params: { path: { credit_name_id: creditNameId } }
          })
        )
        return res.ok ? res.data : undefined
      }
    }))
  ]

  await Promise.all([
    ...tags,
    ...companies,
    eachLimited(singles, CONCURRENCY, async ({ map, id, read }) => {
      const hit = await read(String(id))
      if (hit) {
        map.set(id, label(hit))
      }
    })
  ])
}
