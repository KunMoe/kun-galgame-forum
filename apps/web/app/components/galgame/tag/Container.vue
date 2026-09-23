<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { useRouteQuery } from '@vueuse/router'
import { TAG_FILTER_MAX } from '~/components/search/items'
import type { TagPage, WorkSummary } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { tagItemOf } from '~/utils/galgame/entityCards'

// One page for both lists below: they are mutually exclusive (tag grid while
// nothing is picked, Galgame results once something is), and every selection
// change resets it, so ?page= is never read by the list it was not set on.
const page = usePageQuery()
const tagsLimit = 100
const api = useApiClient()

// The picked tags live in the URL: a multi-tag result is the one thing on this
// page worth sharing, and before this it was a local ref — the link a reader
// copied reopened on the unfiltered tag list.
const tagIdsQuery = useRouteQuery<string>('tag_ids', '', { mode: 'replace' })
const selectedIds = computed<number[]>({
  get: () =>
    tagIdsQuery.value
      .split(',')
      .map(Number)
      .filter((id) => Number.isInteger(id) && id > 0),
  set: (ids) => {
    tagIdsQuery.value = ids.join(',')
  }
})

// Pinned to page 1 while tags are picked, because the tag grid is not on screen
// then: paging the Galgame results would otherwise refetch 100 tags from catalog
// per click, and catalog's limiter counts the whole site as one IP.
const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const { data, status } = await useApi<TagPage>(
  () =>
    `tags:${selectedIds.value.length ? 1 : page.value}:${allowsNsfw.value}`,
  (api) =>
    api.GET('/tags', {
      params: {
        query: {
          page: selectedIds.value.length ? 1 : page.value,
          limit: tagsLimit,
          include_nsfw: allowsNsfw.value
        }
      }
    })
)

const searchResult = ref<GalgameTagItem[]>([])
const searchQuery = ref('')
const isSearching = ref(false)

const displayTags = computed(() =>
  searchQuery.value.trim()
    ? searchResult.value
    : (data.value?.items ?? []).map(tagItemOf)
)

const handleSearch = async () => {
  const q = searchQuery.value.trim()
  if (!q) {
    searchResult.value = []
    return
  }
  isSearching.value = true
  const res = await settle(
    api.GET('/tags', {
      params: { query: { q, limit: 20, include_nsfw: allowsNsfw.value } }
    })
  )
  isSearching.value = false
  searchResult.value = res.ok ? res.data.items.map(tagItemOf) : []
}

watchDebounced(
  () => searchQuery.value,
  () => {
    handleSearch()
  },
  { debounce: 500, maxWait: 1000 }
)

const entityNames = useEntityNames()

// Both writes land in one navigation: useRouteQuery batches every set made in
// the same tick into a single replace.
const setSelectedIds = (ids: number[]) => {
  selectedIds.value = ids
  page.value = 1
}

const toggleTag = (item: SearchEntityItem) => {
  entityNames.remember(item)
  setSelectedIds(
    selectedIds.value.includes(item.id)
      ? selectedIds.value.filter((id) => id !== item.id)
      : [...selectedIds.value, item.id].slice(0, TAG_FILTER_MAX)
  )
}

const chips = computed<FilterChip[]>(() =>
  selectedIds.value.map((id) => ({
    key: String(id),
    label: entityNames.labelOf('tag', id)
  }))
)

const resultWorks = ref<WorkSummary[]>([])
const resultGames = useWorkCards(() => resultWorks.value)
const totalGameCount = ref(0)
const gamesLimit = 24
const loadingGames = ref(false)

const fetchGames = async () => {
  if (!selectedIds.value.length) {
    resultWorks.value = []
    totalGameCount.value = 0
    return
  }
  loadingGames.value = true
  const res = await settle(
    api.GET('/tagged-works', {
      params: {
        query: {
          tag_ids: selectedIds.value.map(String),
          page: page.value,
          limit: gamesLimit,
          include_nsfw: allowsNsfw.value
        }
      }
    })
  )
  loadingGames.value = false
  if (res.ok) {
    resultWorks.value = res.data.items
    totalGameCount.value = res.data.total
  }
}

watch(selectedIds, () => entityNames.resolve({ tag: selectedIds.value }), {
  immediate: true
})

// Immediate, and it must not reset the page: a reader arriving on
// ?tag_ids=…&page=3 has to get page 3.
watch([selectedIds, page], () => fetchGames(), { immediate: true })

const isBrowsing = computed(() => !selectedIds.value.length)
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 标签资料库"
      description="这里展示了绝大多数 Galgame 的标签, 并附带有标签的别名, 您可以点击标签以查看所有含有这个标签的 Galgame"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            默认仅显示了 SFW 的标签, 成人标签既不会出现在列表里, 也搜不到,
            查看它们请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <KunInput
            v-model="searchQuery"
            type="text"
            placeholder="输入以搜索标签, 点击卡片查看该标签下的 Galgame"
          />
        </div>
      </template>
    </KunHeader>

    <FilterBar
      :chips="chips"
      :total="isBrowsing ? (data?.total ?? 0) : totalGameCount"
      :unit="isBrowsing ? '个标签' : '个 Galgame'"
      :pending="isBrowsing ? status === 'pending' : loadingGames"
      @remove="
        setSelectedIds(selectedIds.filter((id) => id !== Number($event)))
      "
      @clear="setSelectedIds([])"
    >
      <FilterEntityMenu
        family="tag"
        icon="lucide:tag"
        label="多标签筛选"
        placeholder="搜索标签, 例如 校园"
        :selected-ids="selectedIds"
        :selected-items="entityNames.itemsOf('tag', selectedIds)"
        multiple
        :max="TAG_FILTER_MAX"
        @toggle="toggleTag"
      />

      <span class="text-default-500 text-sm">
        选中多个标签, 只看同时含有它们的 Galgame
      </span>
    </FilterBar>

    <template v-if="isBrowsing">
      <div
        class="grid grid-cols-2 gap-3 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
      >
        <GalgameTagCard v-for="tag in displayTags" :key="tag.id" :tag="tag" />
      </div>

      <KunLoading v-if="isSearching" />

      <KunNull
        v-else-if="!displayTags.length"
        :description="
          isSfwMode
            ? '没有匹配的标签。成人标签在 SFW 模式下搜不到, 请在设置面板打开 NSFW 开关'
            : undefined
        "
      />

      <KunPagination
        v-if="!searchQuery.trim() && data && data.total > tagsLimit"
        v-model:current-page="page"
        :total-page="Math.ceil(data.total / tagsLimit)"
        :is-loading="status === 'pending'"
      />
    </template>

    <template v-else>
      <KunLoading :loading="loadingGames">
        <GalgameCard v-if="resultGames.length" :galgames="resultGames" />
        <KunNull v-else description="没有同时含有这些标签的 Galgame" />
      </KunLoading>

      <KunPagination
        v-if="totalGameCount > gamesLimit"
        class="mt-3"
        v-model:current-page="page"
        :total-page="Math.ceil(totalGameCount / gamesLimit)"
        :is-loading="loadingGames"
      />
    </template>
  </div>
</template>
