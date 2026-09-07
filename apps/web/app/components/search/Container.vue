<script setup lang="ts">
const route = useRoute()
const router = useRouter()

const { searchHistory } = storeToRefs(usePersistKUNGalgameSearchStore())

// The URL is the only source of truth for what is being searched: the command
// palette, a shared link and the back button all hand the keyword over that way.
const keywords = computed(() => {
  const value = route.query.keywords
  return ((Array.isArray(value) ? value[0] : value) ?? '').trim()
})

const activeType = useTabQuery('all', 'type')
const currentType = computed(() => activeType.value as SearchType)

const setKeywords = (value: string) => {
  if (value === keywords.value) {
    return
  }
  const query = { ...route.query }
  if (value) {
    query.keywords = value
  } else {
    delete query.keywords
  }
  delete query.family
  delete query.page
  router.replace({ query })
}

const rememberHistory = (value: string) => {
  const kept = searchHistory.value.filter((item) => item !== value)
  kept.push(value)
  searchHistory.value = kept.slice(-20)
}

const overview = ref<SearchOverviewResult | null>(null)
// Seeded from the URL rather than false: the immediate watcher below flips it
// synchronously during hydration, so a false here renders a server tree without
// the skeletons the client's first paint has, and every count mismatches.
const overviewPending = ref(!!keywords.value)
const overviewFailed = ref(false)

let latest = 0

// The overview is fetched for every tab, not just 全部: its totals are what the
// category rail counts with, so a deep link straight into 回复 still gets them.
const loadOverview = async (value: string) => {
  const current = ++latest
  if (!value) {
    overview.value = null
    overviewFailed.value = false
    overviewPending.value = false
    return
  }
  overviewPending.value = true
  const data = await kunFetch<SearchOverviewResult>('/search/overview', {
    method: 'GET',
    query: { keywords: value }
  })
  if (current !== latest) {
    return
  }
  overview.value = data
  // kunFetch already popped a toast; without this the content column would just
  // be blank, which reads as "no results" rather than "the request failed".
  overviewFailed.value = !data
  overviewPending.value = false
}

// Nothing awaits this on the server, so an SSR fetch is a request whose answer
// is thrown away and then made again on hydration.
watch(
  keywords,
  (value) => {
    if (!import.meta.server) {
      loadOverview(value)
    }
  },
  { immediate: true }
)

// A lane filter belongs to one lane: 资料库's family and Galgame's own filters
// are meaningless on any other tab, and carrying them across reads as a filter
// the reader cannot see and cannot clear. page goes with them — every lane has
// its own total, so page 7 of 回复 is past the end of 用户.
const LANE_FILTER_KEYS = new Set<string>([
  ...SEARCH_GALGAME_FILTER_KEYS,
  'family',
  'page',
  'type'
])

const setType = (value: SearchType) => {
  const query = Object.fromEntries(
    Object.entries(route.query).filter(([key]) => !LANE_FILTER_KEYS.has(key))
  )
  if (value !== 'all') {
    query.type = value
  }
  router.replace({ query })
}
</script>

<template>
  <div class="min-h-[calc(100dvh-6rem)] space-y-6">
    <KunHeader
      name="搜索"
      description="一次搜索整个论坛: Galgame, 话题, 资料库中的角色 / 会社 / Staff / 标签, Galgame 下载资源, 用户, 回复与评论。"
    >
      <template #endContent>
        <div class="text-default-500 text-sm">
          搜索结果一并包含 NSFW 的 Galgame; 未开启 NSFW 时, 资料库中的成人标签,
          以及 R18 游戏的下载资源会被隐藏。按厂商 / 会社 / 多标签精确筛选请前往
          <KunLink to="/galgame/official">Galgame 会社资料库</KunLink>
          或者
          <KunLink to="/galgame/tag">Galgame 标签资料库</KunLink>。
        </div>
      </template>
    </KunHeader>

    <!--
      The box follows the reader down the page instead of scrolling away: on a
      results page the next thing they want is almost always a different query.
      Its own background is opaque because the rail slides underneath it.
    -->
    <div
      class="bg-background/90 lg:sticky lg:top-[68px] lg:z-20 lg:-mx-2 lg:px-2 lg:py-2 lg:backdrop-blur"
    >
      <SearchBox
        :keywords="keywords"
        @submit="setKeywords"
        @remember="rememberHistory"
      />
    </div>

    <SearchHistory v-if="!keywords" @select="setKeywords" />

    <div v-else class="gap-8 lg:grid lg:grid-cols-[13rem_minmax(0,1fr)]">
      <div class="mb-6 lg:mb-0">
        <div class="lg:sticky lg:top-[9.5rem]">
          <SearchNav
            :model-value="currentType"
            :totals="overview?.totals ?? null"
            :pending="overviewPending"
            @update:model-value="setType"
          />
        </div>
      </div>

      <div class="min-w-0">
        <SearchOverview
          v-if="currentType === 'all'"
          :keywords="keywords"
          :overview="overview"
          :pending="overviewPending"
          :failed="overviewFailed"
          @open="setType"
        />

        <SearchEntities
          v-else-if="currentType === 'entity'"
          :keywords="keywords"
        />

        <SearchList
          v-else
          :key="currentType"
          :keywords="keywords"
          :type="currentType as SearchPagedType"
        />
      </div>
    </div>
  </div>
</template>
