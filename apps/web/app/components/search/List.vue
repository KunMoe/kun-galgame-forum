<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import { SEARCH_CATEGORY_MAP } from './items'

const props = defineProps<{
  keywords: string
  type: SearchPagedType
}>()

const PAGE_SIZE = 24

interface SearchPage {
  items: SearchResult[]
  total: number
}

const results = ref<SearchResult[]>([])
const total = ref(0)
const pending = ref(!!props.keywords)
const failed = ref(false)
// In the URL, so a reload and a shared link both land on the page the reader
// was actually on. `replace`, like every other paginated page here, so paging
// does not fill the history. SearchContainer drops it when the category or the
// keyword changes, SearchGalgameFilter when a filter does.
const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const top = useTemplateRef<HTMLElement>('top')

const meta = computed(() => SEARCH_CATEGORY_MAP[props.type])
const totalPage = computed(() => Math.ceil(total.value / PAGE_SIZE))

const isGalgame = computed(() => props.type === 'galgame')

const route = useRoute()
const galgameFilter = computed(() =>
  isGalgame.value
    ? Object.fromEntries(
        SEARCH_GALGAME_FILTER_KEYS.map((key) => [key, route.query[key] ?? ''])
      )
    : {}
)

// route.query is a new object on every navigation, so galgameFilter is too —
// watching it directly reloaded the lane on any query change, page included.
// The string only changes when a filter actually does.
const filterKey = computed(() => JSON.stringify(galgameFilter.value))

// The rail counts with the overview, which is never filtered, so a filtered
// lane sits under a rail that still says 2261 while the page says 0. Naming the
// filter in the empty state is what stops that reading as a broken count.
const isFiltered = computed(() =>
  SEARCH_GALGAME_FILTER_KEYS.some(
    (key) => key !== 'sort' && !!galgameFilter.value[key]
  )
)

let latest = 0

const fetchPage = (target: number) =>
  props.type === 'toolset'
    ? kunFetch<SearchPage>('/toolset', {
        method: 'GET',
        query: { query: props.keywords, page: target, limit: PAGE_SIZE }
      })
    : kunFetch<SearchPage>('/search', {
        method: 'GET',
        query: {
          keywords: props.keywords,
          type: props.type,
          page: target,
          limit: PAGE_SIZE,
          ...galgameFilter.value
        }
      })

const load = async () => {
  const current = ++latest
  if (!props.keywords) {
    results.value = []
    total.value = 0
    pending.value = false
    return
  }
  pending.value = true
  const data = await fetchPage(page.value)
  if (current !== latest) {
    return
  }
  // kunFetch already popped a toast; telling the reader "nothing found" when the
  // request never came back is the one thing this must not do.
  failed.value = !data
  total.value = data?.total ?? 0

  // A link outlives the result set it was copied from. ?page=16 against a lane
  // that has since shrunk to 10 pages rendered "杂鱼杂鱼杂鱼~什么也没有搜索到"
  // under a header still reading 共 364 个话题, with no page marked current.
  const lastPage = Math.max(1, Math.ceil(total.value / PAGE_SIZE))
  if (page.value > lastPage) {
    page.value = lastPage
    return
  }

  results.value = data?.items ?? []
  pending.value = false
}

watch([() => props.keywords, filterKey], () => {
  results.value = []
})

watch(
  [() => props.keywords, filterKey, page],
  async (_now, before) => {
    await load()
    // The paginator sits below a full page of results, so a page opened from it
    // would otherwise start scrolled past its own first row. Only a page change
    // earns the scroll — a new keyword or filter already starts at the top.
    if (before && before[2] !== page.value) {
      top.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  },
  { immediate: true }
)
</script>

<template>
  <div ref="top" class="scroll-mt-40 space-y-6">
    <SearchGalgameFilter v-if="isGalgame" :total="total" :pending="pending" />

    <p v-else class="text-default-500 text-sm">
      <template v-if="pending && !results.length">正在搜索…</template>
      <template v-else-if="total">
        共 <span class="text-default-700 tabular-nums">{{ total }}</span>
        {{ meta.countUnit }}
      </template>
    </p>

    <SearchSkeleton
      v-if="pending && !results.length"
      :shape="type === 'galgame' || type === 'toolset' ? 'card' : 'row'"
    />

    <KunLoading v-else-if="results.length" :loading="pending">
      <SearchResult :results="results" :type="type" :keywords="keywords" />
    </KunLoading>

    <KunNull v-else-if="failed" description="搜索没能完成, 请稍后重试" />

    <KunNull
      v-else-if="keywords"
      :description="
        isFiltered
          ? '这些筛选条件下没有 Galgame, 试着去掉一两个'
          : '杂鱼杂鱼杂鱼~什么也没有搜索到'
      "
    />

    <KunPagination
      v-if="totalPage > 1"
      v-model:current-page="page"
      :total-page="totalPage"
      :is-loading="pending"
    />
  </div>
</template>
