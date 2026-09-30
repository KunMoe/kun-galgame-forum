import {
  computed,
  nextTick,
  type ComputedRef,
  type WritableComputedRef
} from 'vue'
import type { LocationQuery, Router } from 'vue-router'

export interface QueryField<T> {
  parse: (raw: string | undefined) => T
  format: (value: T) => string
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type QuerySchema = Record<string, QueryField<any>>

export type QueryValues<S extends QuerySchema> = {
  [K in keyof S]: S[K] extends QueryField<infer T> ? T : never
}

export type QueryState<S extends QuerySchema> = {
  [K in keyof S]: WritableComputedRef<QueryValues<S>[K]>
} & {
  state: ComputedRef<QueryValues<S>>
  set: (patch: Partial<QueryValues<S>>) => void
}

export interface QueryRoute {
  readonly path: string
  readonly hash: string
  readonly query: LocationQuery
}

type QueryPatch = Record<string, string | undefined>

export const queryString = (fallback = ''): QueryField<string> => ({
  parse: (raw) => raw || fallback,
  format: (value) => value
})

export const queryInt = (
  fallback = 0,
  min = Number.MIN_SAFE_INTEGER
): QueryField<number> => ({
  parse: (raw) => {
    const value = Number(raw)
    return raw && Number.isInteger(value) && value >= min ? value : fallback
  },
  format: String
})

export const queryPage = () => queryInt(1, 1)

export const queryEnum = <T extends string>(
  values: readonly T[],
  fallback: T,
  aliases: Record<string, T> = {}
): QueryField<T> => ({
  parse: (raw) => {
    const value = raw === undefined ? undefined : (aliases[raw] ?? raw)
    return values.includes(value as T) ? (value as T) : fallback
  },
  format: (value) => value
})

export const queryList = <T extends string>(
  pick: (item: string) => T | undefined
): QueryField<T[]> => {
  const parse = (raw: string | undefined) => [
    ...new Set(
      (raw ?? '')
        .split(',')
        .map(pick)
        .filter((item): item is T => !!item)
    )
  ]
  return { parse, format: (values) => values.join(',') }
}

export const queryIds = (): QueryField<number[]> => ({
  parse: (raw) => [
    ...new Set(
      (raw ?? '')
        .split(',')
        .map(Number)
        .filter((id) => Number.isInteger(id) && id > 0)
    )
  ],
  format: (ids) => ids.join(',')
})

const firstValue = (value: LocationQuery[string] | undefined) =>
  (Array.isArray(value) ? value[0] : value) ?? undefined

const queued = new WeakMap<Router, QueryPatch>()
const inFlight = new WeakMap<Router, LocationQuery>()

// Every write on a page lands in one replace per tick, built on the query the
// last replace asked for rather than the one the route still shows: two
// replaces a tick apart each started from the old query, and the second undid
// the first. A write from a page that is already leaving is dropped, or it
// would carry that page's query into the next page's URL.
export const writeQuery = (
  router: Router,
  route: QueryRoute,
  patch: QueryPatch
) => {
  const batch = queued.get(router)
  if (batch) {
    Object.assign(batch, patch)
    return
  }
  queued.set(router, { ...patch })
  nextTick(() => {
    const merged = queued.get(router) ?? {}
    queued.delete(router)
    if (router.currentRoute.value.path !== route.path) {
      return
    }
    const query = Object.fromEntries(
      Object.entries({
        ...(inFlight.get(router) ?? route.query),
        ...merged
      }).filter(([, value]) => value !== undefined)
    ) as LocationQuery
    inFlight.set(router, query)
    router
      .replace({ path: route.path, query, hash: route.hash })
      .finally(() => {
        if (inFlight.get(router) === query) {
          inFlight.delete(router)
        }
      })
  })
}

// One parse of the whole query per navigation. With a ref per key, a
// navigation reached the list's fetch key one key at a time; useAsyncData
// watches its key synchronously, so every half-applied step (page already 1,
// the new platform not yet in) went out as a request of its own. A key whose
// value did not change keeps its previous value, so nothing downstream reruns
// for it.
export const createQueryState = <S extends QuerySchema>(
  schema: S,
  options: { route: QueryRoute; router: Router; pageKey?: keyof S & string }
): QueryState<S> => {
  const { route, router, pageKey } = options
  const keys = Object.keys(schema) as (keyof S & string)[]
  const textOf = (key: keyof S & string, value: unknown) =>
    schema[key]!.format(value)
  const defaults = Object.fromEntries(
    keys.map((key) => [key, textOf(key, schema[key]!.parse(undefined))])
  )

  const state = computed<QueryValues<S>>((previous) => {
    let changed = !previous
    const next = {} as QueryValues<S>
    for (const key of keys) {
      const value = schema[key]!.parse(firstValue(route.query[key]))
      if (previous && textOf(key, previous[key]) === textOf(key, value)) {
        next[key] = previous[key]
      } else {
        next[key] = value
        changed = true
      }
    }
    return changed || !previous ? next : previous
  })

  // Moving to another filter sends the list back to its first page here, at
  // the write, and not in a watcher on the values: a watcher also fired when
  // a link or back/forward brought a new filter, and threw away the ?page=
  // that came with it.
  const set = (patch: Partial<QueryValues<S>>) => {
    const out: QueryPatch = {}
    let movesAway = false
    for (const key of Object.keys(patch) as (keyof S & string)[]) {
      const text = textOf(key, patch[key])
      out[key] = text === '' || text === defaults[key] ? undefined : text
      if (key !== pageKey && text !== textOf(key, state.value[key])) {
        movesAway = true
      }
    }
    if (pageKey && movesAway && !(pageKey in patch)) {
      out[pageKey] = undefined
    }
    writeQuery(router, route, out)
  }

  const refs = Object.fromEntries(
    keys.map((key) => [
      key,
      computed({
        get: () => state.value[key],
        set: (value) => set({ [key]: value } as Partial<QueryValues<S>>)
      })
    ])
  )

  return { ...refs, state, set } as QueryState<S>
}
