import { settle, type ClientProblem } from '#shared/utils/api/problem'
import type { ApiClient } from '#shared/utils/api/client'
import { isHistoryPop } from '~/utils/historyPop'
import { useApiClient, useKunApi } from './useKunApi'

type Page<Item> = { items: Item[]; next_cursor?: string; total?: number }

type Head<P> = Omit<P, 'items' | 'next_cursor' | 'total'>

type Snapshot<Item> = {
  items: Item[]
  nextCursor: string | undefined
  total: number | undefined
  head: Record<string, unknown>
}

type FetchPage<Item, P extends Page<Item>> = (
  api: ApiClient,
  cursor: string | undefined,
  ctx: { signal?: AbortSignal }
) => Promise<{ data?: P; error?: unknown; response: Response }>

type UseCursorListReturn<Item, P extends Page<Item>> = {
  items: Ref<Item[]>
  hasMore: Ref<boolean>
  total: Ref<number | undefined>
  // What the first page carries besides its items, such as facet counts.
  head: Ref<Head<P> | undefined>
  problem: ComputedRef<ClientProblem | null>
  pending: ComputedRef<boolean>
  status: ReturnType<typeof useAsyncData>['status']
  loadingMore: Ref<boolean>
  loadMore: () => Promise<void>
  refresh: ReturnType<typeof useAsyncData>['refresh']
}

const SNAPSHOT_LIMIT = 10
// Snapshots are a client Map: a server module Map is shared across requests,
// and the payload only holds the SSR first page.
const snapshots = new Map<string, Snapshot<unknown>>()

const rememberSnapshot = <Item>(key: string, snapshot: Snapshot<Item>) => {
  if (!import.meta.client) {
    return
  }
  snapshots.delete(key)
  snapshots.set(key, snapshot)
  if (snapshots.size > SNAPSHOT_LIMIT) {
    const oldest = snapshots.keys().next().value
    if (oldest !== undefined) {
      snapshots.delete(oldest)
    }
  }
}

export const useCursorList = <
  Item extends { id: string },
  P extends Page<Item> = Page<Item>
>(
  key: MaybeRefOrGetter<string>,
  fetchPage: FetchPage<Item, P>
): UseCursorListReturn<Item, P> & Promise<UseCursorListReturn<Item, P>> => {
  const api = useApiClient()
  const items = ref<Item[]>([]) as Ref<Item[]>
  const hasMore = ref(false)
  const total = ref<number | undefined>(undefined)
  const head = ref<Head<P> | undefined>(undefined) as Ref<Head<P> | undefined>
  const nextCursor = ref<string | undefined>(undefined)
  const loadingMore = ref(false)
  let loadMoreGeneration = 0
  let seenKey: string | undefined

  const listKey = () => `cursor:${toValue(key)}`

  const list = useKunApi(
    listKey,
    (client, { signal }) => fetchPage(client, undefined, { signal }),
    {
      getCachedData: (dataKey, nuxtApp, ctx) => {
        if (nuxtApp.isHydrating) {
          return nuxtApp.payload.data[dataKey]
        }
        if (!import.meta.client || ctx.cause !== 'initial' || !isHistoryPop()) {
          return undefined
        }
        const snap = snapshots.get(listKey()) as Snapshot<Item> | undefined
        if (!snap) {
          return undefined
        }
        return {
          ok: true as const,
          data: {
            ...snap.head,
            items: snap.items,
            ...(snap.nextCursor ? { next_cursor: snap.nextCursor } : {}),
            ...(snap.total === undefined ? {} : { total: snap.total })
          } as P
        }
      }
    }
  )

  const applyPage = (page: P, dataKey: string) => {
    const {
      items: pageItems,
      next_cursor: pageCursor,
      total: pageTotal,
      ...rest
    } = page
    items.value = pageItems
    nextCursor.value = pageCursor
    hasMore.value = Boolean(pageCursor)
    total.value = pageTotal
    head.value = rest as Head<P>
    rememberSnapshot(dataKey, {
      items: pageItems,
      nextCursor: pageCursor,
      total: pageTotal,
      head: rest
    })
  }

  watch(
    () =>
      ({
        dataKey: listKey(),
        data: list.data.value,
        problem: list.problem.value,
        status: list.status.value
      }) as const,
    (curr) => {
      if (seenKey !== undefined && seenKey !== curr.dataKey) {
        loadMoreGeneration++
        loadingMore.value = false
        if (curr.status === 'pending') {
          items.value = []
          hasMore.value = false
          nextCursor.value = undefined
          total.value = undefined
          head.value = undefined
          seenKey = curr.dataKey
          return
        }
      }
      seenKey = curr.dataKey
      if (curr.data) {
        applyPage(curr.data, curr.dataKey)
        return
      }
      if (curr.problem) {
        items.value = []
        hasMore.value = false
        nextCursor.value = undefined
        total.value = undefined
        head.value = undefined
      }
    },
    { immediate: true, flush: 'sync' }
  )

  const loadMore = async () => {
    if (loadingMore.value || !hasMore.value) {
      return
    }
    const generation = loadMoreGeneration
    const dataKey = listKey()
    const cursor = nextCursor.value
    if (!cursor) {
      return
    }
    loadingMore.value = true
    const result = await settle(fetchPage(api, cursor, {}))
    if (generation !== loadMoreGeneration) {
      return
    }
    loadingMore.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    const seen = new Set(items.value.map((item) => item.id))
    const appended = result.data.items.filter((item) => !seen.has(item.id))
    const merged = [...items.value, ...appended]
    items.value = merged
    nextCursor.value = result.data.next_cursor
    hasMore.value = Boolean(result.data.next_cursor)
    total.value = result.data.total ?? total.value
    rememberSnapshot(dataKey, {
      items: merged,
      nextCursor: result.data.next_cursor,
      total: total.value,
      head: { ...head.value }
    })
  }

  const result: UseCursorListReturn<Item, P> = {
    items,
    hasMore,
    total,
    head,
    problem: list.problem,
    pending: computed(
      () => list.status.value === 'pending' && items.value.length === 0
    ),
    status: list.status,
    loadingMore,
    loadMore,
    refresh: list.refresh
  }

  return Object.assign(
    Promise.resolve(list).then(() => result),
    result
  )
}
