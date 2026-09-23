import { settle, type ClientProblem } from '#shared/utils/api/problem'
import type { ApiClient } from '#shared/utils/api/client'
import { isHistoryPop } from '~/utils/historyPop'
import { useApiClient } from './useApi'

type Page<Item> = { items: Item[]; next_cursor?: string; total?: number }

type Snapshot<Item> = {
  items: Item[]
  nextCursor: string | undefined
  total: number | undefined
}

type FetchPage<Item> = (
  api: ApiClient,
  cursor: string | undefined,
  ctx: { signal?: AbortSignal }
) => Promise<{ data?: Page<Item>; error?: unknown; response: Response }>

type UseCursorListReturn<Item> = {
  items: Ref<Item[]>
  hasMore: Ref<boolean>
  total: Ref<number | undefined>
  problem: ComputedRef<ClientProblem | null>
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

export const useCursorList = <Item extends { id: string }>(
  key: MaybeRefOrGetter<string>,
  fetchPage: FetchPage<Item>
): UseCursorListReturn<Item> & Promise<UseCursorListReturn<Item>> => {
  const api = useApiClient()
  const items = ref<Item[]>([]) as Ref<Item[]>
  const hasMore = ref(false)
  const total = ref<number | undefined>(undefined)
  const nextCursor = ref<string | undefined>(undefined)
  const loadingMore = ref(false)
  let loadMoreGeneration = 0
  let seenKey: string | undefined

  const asyncKey = () => `cursor:${toValue(key)}`

  const asyncData = useAsyncData(
    asyncKey,
    (_app, { signal }) => settle(fetchPage(api, undefined, { signal })),
    {
      getCachedData: (dataKey, nuxtApp, ctx) => {
        if (nuxtApp.isHydrating) {
          return nuxtApp.payload.data[dataKey]
        }
        if (!import.meta.client || ctx.cause !== 'initial' || !isHistoryPop()) {
          return undefined
        }
        const snap = snapshots.get(dataKey) as Snapshot<Item> | undefined
        if (!snap) {
          return undefined
        }
        return {
          ok: true as const,
          data: {
            items: snap.items,
            ...(snap.nextCursor ? { next_cursor: snap.nextCursor } : {}),
            ...(snap.total === undefined ? {} : { total: snap.total })
          }
        }
      }
    }
  )

  const applyPage = (page: Page<Item>, dataKey: string) => {
    items.value = page.items
    nextCursor.value = page.next_cursor
    hasMore.value = Boolean(page.next_cursor)
    total.value = page.total
    rememberSnapshot(dataKey, {
      items: page.items,
      nextCursor: page.next_cursor,
      total: page.total
    })
  }

  watch(
    () =>
      ({
        dataKey: asyncKey(),
        result: asyncData.data.value,
        status: asyncData.status.value
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
          seenKey = curr.dataKey
          return
        }
      }
      seenKey = curr.dataKey
      if (curr.result?.ok) {
        applyPage(curr.result.data, curr.dataKey)
        return
      }
      if (curr.result && !curr.result.ok) {
        items.value = []
        hasMore.value = false
        nextCursor.value = undefined
        total.value = undefined
      }
    },
    { immediate: true, flush: 'sync' }
  )

  const problem = computed(() =>
    asyncData.data.value && !asyncData.data.value.ok
      ? asyncData.data.value.problem
      : null
  )

  const loadMore = async () => {
    if (loadingMore.value || !hasMore.value) {
      return
    }
    const generation = loadMoreGeneration
    const dataKey = asyncKey()
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
      total: total.value
    })
  }

  const result: UseCursorListReturn<Item> = {
    items,
    hasMore,
    total,
    problem,
    status: asyncData.status,
    loadingMore,
    loadMore,
    refresh: asyncData.refresh
  }

  return Object.assign(
    Promise.resolve(asyncData).then(() => result),
    result
  )
}
