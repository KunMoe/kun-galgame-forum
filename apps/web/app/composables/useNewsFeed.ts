import { useIntersectionObserver, useThrottleFn } from '@vueuse/core'
import { settle } from '#shared/utils/api/problem'

interface KunNewsFeedOptions {
  limit?: number
  maxAutoLoads?: number
  lane?: Ref<string>
  source?: Ref<string>
  year?: Ref<number>
  month?: Ref<number>
}

const asFilter = (value?: string) =>
  !value || value === 'all' ? undefined : value

const asNumber = (value?: number) => (value && value > 0 ? value : undefined)

export const useNewsFeed = async (options: KunNewsFeedOptions = {}) => {
  const limit = options.limit ?? 20
  const maxAutoLoads = options.maxAutoLoads ?? 4

  const lane = computed(() => asFilter(options.lane?.value))
  const source = computed(() => asFilter(options.source?.value))
  const year = computed(() => asNumber(options.year?.value))
  const month = computed(() =>
    year.value ? asNumber(options.month?.value) : undefined
  )

  const items = ref<KunNewsItem[]>([])
  const total = ref(0)
  const cursor = ref('')
  const hasMore = ref(true)
  const isLoadingMore = ref(false)
  const autoLoadCount = ref(0)
  let controller: AbortController | null = null

  // A page in flight belongs to the filter that asked for it. Bumping the
  // generation on every first page drops a late reply instead of appending
  // 月幕's items under a 批评-only filter.
  let generation = 0

  const api = useApiClient()
  const query = computed(() => ({
    limit,
    include_total: true,
    ...(lane.value ? { lane: lane.value as KunNewsItem['lane'] } : {}),
    ...(source.value ? { news_source: source.value } : {}),
    ...(year.value ? { year: year.value } : {}),
    ...(month.value ? { month: month.value } : {})
  }))

  // Everything that registers on the component instance or its effect scope —
  // onBeforeUnmount, useIntersectionObserver, watch — has to run before this
  // function's first await. Vue only restores the instance across a top-level
  // await in <script setup>, never inside a composable, so a hook moved down
  // here logs "onBeforeUnmount is called when there is no active component
  // instance" and silently never fires.
  const sourcesReady = useNewsSources()
  const { byKey: sources } = sourcesReady
  const asyncData = useApi(
    () => `news-items:${JSON.stringify(query.value)}`,
    (client, { signal }) =>
      client.GET('/news-items', { params: { query: query.value }, signal })
  )
  const { data, status, problem: error } = asyncData

  type NewsPage = NonNullable<typeof data.value>

  const applyPage = (page: NewsPage) => {
    generation++
    items.value = page.items
    total.value = page.total ?? 0
    cursor.value = page.next_cursor ?? ''
    hasMore.value = !!page.next_cursor
    isLoadingMore.value = false
    autoLoadCount.value = 0
  }

  watch(data, (page) => {
    if (page) applyPage(page)
  })

  const loadMore = async (auto = false) => {
    if (isLoadingMore.value || !hasMore.value || !cursor.value) return
    if (auto) {
      if (autoLoadCount.value >= maxAutoLoads) return
      autoLoadCount.value++
    } else {
      autoLoadCount.value = 0
    }
    const gen = generation
    isLoadingMore.value = true
    controller = new AbortController()
    const result = await settle(
      api.GET('/news-items', {
        params: { query: { ...query.value, cursor: cursor.value } },
        signal: controller.signal
      })
    )
    if (gen !== generation) return
    isLoadingMore.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    items.value.push(...result.data.items)
    cursor.value = result.data.next_cursor ?? ''
    hasMore.value = !!result.data.next_cursor
  }

  const autoLoad = useThrottleFn(() => loadMore(true), 600)
  onBeforeUnmount(() => controller?.abort())

  const sentinel = ref<HTMLElement | null>(null)
  useIntersectionObserver(
    sentinel,
    ([entry]) => {
      if (entry?.isIntersecting) autoLoad()
    },
    { rootMargin: '150px' }
  )

  const groups = computed(() => groupNewsItems(items.value, sources.value))

  // The watcher above never fires on the server — Vue skips non-immediate
  // watchers during SSR — so the first page has to be applied by hand, or the
  // rendered HTML ships empty and the list only appears after hydration.
  await Promise.all([asyncData, sourcesReady])
  if (data.value) applyPage(data.value)

  return {
    filters: { lane, source, year, month },
    items,
    sources,
    groups,
    total,
    status,
    error,
    hasMore,
    isLoadingMore,
    loadMore,
    sentinel
  }
}
