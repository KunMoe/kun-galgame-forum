// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { createApiClient } from '#shared/utils/api/client'
import type { CatalogName } from '#shared/utils/catalogName'
import { resolveEditNames, type EditNameMaps } from './resolveEditNames'

const origin = 'https://forum.test'

const entity = (id: string, name: string) => ({
  id,
  display_name: name,
  latin: null,
  localized: {}
})

const json = (status: number, body: unknown, type = 'application/json') =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': type }
  })

const notFound = () =>
  json(
    404,
    {
      type: 'https://developer.nextmoe.dev/problems/platform/not-found',
      title: 'Not found',
      status: 404,
      code: 'NOT_FOUND',
      request_id: 'req_01M00000000000000000000000',
      errors: []
    },
    'application/problem+json'
  )

const nameOf = (n: CatalogName) => ({
  name: n.display_name,
  original: n.display_name
})

const emptyMaps = (): EditNameMaps => ({
  tag: new Map(),
  official: new Map(),
  engine: new Map(),
  series: new Map(),
  character: new Map(),
  staff: new Map()
})

const fakeApi = () => {
  const requests: string[] = []
  let inFlight = 0
  let peak = 0
  const api = createApiClient({
    origin,
    fetch: async (input) => {
      const url = new URL((input as Request).url)
      const [, , version, ...rest] = url.pathname.split('/')
      const path = version === 'v1' ? rest.join('/') : `legacy:${url.pathname}`
      requests.push(path + url.search)
      const single = path !== 'tags' && path !== 'companies'
      if (single) {
        inFlight++
        peak = Math.max(peak, inFlight)
      }
      await new Promise((resolve) => setTimeout(resolve, 5))
      if (single) {
        inFlight--
      }
      const ids = url.searchParams.get('ids')?.split(',') ?? []
      switch (true) {
        case path === 'tags':
          return json(200, {
            object: 'list',
            items: ids
              .filter((id) => id !== '404')
              .map((id) => ({ ...entity(id, `tag ${id}`), object: 'tag' })),
            total: ids.length,
            total_relation: 'eq'
          })
        case path === 'companies':
          return json(200, {
            object: 'list',
            items: ids.map((id) => entity(id, `company ${id}`)),
            total: ids.length,
            total_relation: 'eq'
          })
        case path.startsWith('series/'):
          return notFound()
        case path.startsWith('credit-names/'):
          return json(503, { oops: true })
        default: {
          const [family, id = ''] = path.split('/')
          return json(200, entity(id, `${family} ${id}`))
        }
      }
    }
  })
  return { api, requests, peak: () => peak }
}

describe('resolveEditNames', () => {
  it('reads each family through its v1 face and leaves misses out quietly', async () => {
    const { api, requests } = fakeApi()
    const maps = emptyMaps()
    maps.engine.set(7, 'already known')

    await resolveEditNames(
      api,
      {
        tag: [1, 404],
        official: [5],
        engine: [7, 8],
        series: [9],
        character: [11, 11, 12],
        staff: [13]
      },
      nameOf,
      maps
    )

    expect(maps.tag).toEqual(new Map([[1, 'tag 1']]))
    expect(maps.official).toEqual(new Map([[5, 'company 5']]))
    expect(maps.engine).toEqual(
      new Map([
        [7, 'already known'],
        [8, 'engines 8']
      ])
    )
    expect(maps.series.size).toBe(0)
    expect(maps.character).toEqual(
      new Map([
        [11, 'characters 11'],
        [12, 'characters 12']
      ])
    )
    expect(maps.staff.size).toBe(0)

    const query = (prefix: string) => {
      const hit = requests.find((r) => r.startsWith(prefix))
      return Object.fromEntries(new URL(hit ?? '', origin).searchParams)
    }
    expect(query('tags')).toEqual({
      ids: '1,404',
      include_nsfw: 'true',
      limit: '100'
    })
    expect(query('companies')).toEqual({ ids: '5', limit: '100' })
    expect(query('characters/11')).toEqual({ include_nsfw: 'true' })
    expect(requests.filter((r) => r.startsWith('characters/11'))).toHaveLength(
      1
    )
    expect(requests.some((r) => r.startsWith('engines/7'))).toBe(false)
    expect(requests.some((r) => r.startsWith('legacy:'))).toBe(false)
  })

  it('batches tags by 100 and keeps single reads to four at a time', async () => {
    const { api, requests, peak } = fakeApi()
    const maps = emptyMaps()
    const many = Array.from({ length: 150 }, (_, i) => i + 1)

    await resolveEditNames(
      api,
      {
        tag: many,
        official: [],
        engine: many.slice(0, 20),
        series: [],
        character: [],
        staff: []
      },
      nameOf,
      maps
    )

    expect(requests.filter((r) => r.startsWith('tags'))).toHaveLength(2)
    expect(maps.tag.size).toBe(150)
    expect(maps.engine.size).toBe(20)
    expect(peak()).toBe(4)
  })

  it('does nothing when every id already has a name', async () => {
    const { api, requests } = fakeApi()
    const maps = emptyMaps()
    maps.tag.set(1, 'known')
    await resolveEditNames(
      api,
      {
        tag: [1],
        official: [],
        engine: [],
        series: [],
        character: [],
        staff: []
      },
      nameOf,
      maps
    )
    expect(requests).toHaveLength(0)
  })
})
