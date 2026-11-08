<script setup lang="ts">
import type { BreadcrumbList, CollectionPage, WithContext } from 'schema-dts'
import type {
  CollectionSummary,
  ListCollectionTagCount
} from '#shared/utils/api/schemas'
import { useCursorList } from '~/composables/useCursorList'

const SORT_VALUES = COLLECTION_DIRECTORY_SORTS.map((s) => s.value)

const { sort, work, tag, q } = useQueryState({
  sort: queryEnum(SORT_VALUES, 'followers_desc'),
  work: queryString(),
  tag: queryString(),
  q: queryString()
})

const searchDraft = ref(q.value)
watch(q, (value) => {
  searchDraft.value = value
})
const submitSearch = () => {
  q.value = searchDraft.value.trim().slice(0, 50)
}
const toggleTag = (value: string) => {
  tag.value = tag.value === value ? '' : value
}

const { allowsNsfw, stanceKey } = useContentStance()
const SORT_SELECT_OPTIONS = COLLECTION_DIRECTORY_SORTS.map((s) => ({
  value: s.value,
  label: s.textValue
}))

const followedOpen = ref(false)
const followedMounted = ref(false)
const { count: followedUpdates, refresh: refreshFollowedUpdates } =
  useFollowedCollectionUpdates()
onMounted(() => {
  void refreshFollowedUpdates()
})
const openFollowed = () => {
  if (requireLogin()) {
    followedMounted.value = true
    followedOpen.value = true
  }
}

const { data: tagCloud } = useKunApi<ListCollectionTagCount>(
  () => `collection-tags:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/collection-tags', {
      params: { query: { include_nsfw: allowsNsfw.value, limit: 24 } },
      signal
    })
)
const workId = computed(() =>
  /^[1-9]\d{0,18}$/.test(work.value) ? work.value : ''
)

const { items, hasMore, pending, loadingMore, loadMore, problem } =
  useCursorList<CollectionSummary>(
    () =>
      `collection-directory:${sort.value}:${workId.value}:${tag.value}:${q.value}:${stanceKey.value}`,
    (api, cursor, { signal }) =>
      api.GET('/collections', {
        params: {
          query: {
            sort: sort.value,
            include_nsfw: allowsNsfw.value,
            limit: 24,
            ...(workId.value ? { contains_work_id: workId.value } : {}),
            ...(tag.value ? { tag: tag.value } : {}),
            ...(q.value ? { q: q.value } : {}),
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )

const pageUrl = `${kungal.domain.main}/collection`
const seoDescription =
  '浏览鲲 Galgame 玩家整理的 Galgame 收藏夹：按主题、作者、剧本家或题材分类的视觉小说清单与推荐，可以关注喜欢的收藏夹并参与讨论。'

useHead(() => ({
  script: [
    {
      id: 'schema-org-collection-directory',
      type: 'application/ld+json',
      innerHTML: {
        '@context': 'https://schema.org',
        '@type': 'CollectionPage',
        name: 'Galgame 收藏夹',
        description: seoDescription,
        url: pageUrl,
        inLanguage: 'zh-CN',
        mainEntity: {
          '@type': 'ItemList',
          itemListElement: items.value.slice(0, 24).map((c, i) => ({
            '@type': 'ListItem',
            position: i + 1,
            url: `${kungal.domain.main}/collection/${c.id}`,
            name: collectionDisplayName(c, c.owner.name)
          }))
        }
      } satisfies WithContext<CollectionPage>
    },
    {
      id: 'schema-org-breadcrumb',
      type: 'application/ld+json',
      innerHTML: {
        '@context': 'https://schema.org',
        '@type': 'BreadcrumbList',
        itemListElement: [
          {
            '@type': 'ListItem',
            position: 1,
            name: '首页',
            item: kungal.domain.main
          },
          {
            '@type': 'ListItem',
            position: 2,
            name: 'Galgame 收藏夹',
            item: pageUrl
          }
        ]
      } satisfies WithContext<BreadcrumbList>
    }
  ]
}))

useKunSeoMeta(() =>
  workId.value || tag.value || q.value
    ? { noindex: true, title: 'Galgame 收藏夹' }
    : {
        title: 'Galgame 收藏夹 - 玩家整理的 Galgame 清单与推荐',
        description: seoDescription,
        keywords: 'Galgame 收藏夹,Galgame 推荐,视觉小说清单,Galgame 清单'
      }
)
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 收藏夹"
      description="玩家们整理的 Galgame 清单。公开、有名字、收录至少 3 部作品的收藏夹会出现在这里"
    />

    <div class="flex flex-wrap items-center gap-3">
      <form
        class="min-w-48 flex-1"
        role="search"
        @submit.prevent="submitSearch"
      >
        <KunInput
          v-model="searchDraft"
          type="search"
          placeholder="搜索收藏夹名称、简介或标签"
          aria-label="搜索收藏夹"
          :maxlength="50"
          @blur="submitSearch"
        >
          <template #prefix>
            <KunIcon name="lucide:search" class="text-default-400" />
          </template>
        </KunInput>
      </form>
      <KunSelect
        v-model="sort"
        :options="SORT_SELECT_OPTIONS"
        class-name="w-40"
        aria-label="排序"
      />
      <KunButton
        variant="flat"
        size="sm"
        class-name="relative"
        @click="openFollowed"
      >
        <KunIcon name="lucide:bookmark-check" />
        我关注的收藏夹
        <span
          v-if="followedUpdates > 0"
          class="bg-danger absolute -top-1 -right-1 min-w-5 rounded-full px-1 text-center text-xs leading-5 text-white"
          :aria-label="`${followedUpdates} 个收藏夹有新作品`"
        >
          {{ followedUpdates >= 100 ? '99+' : followedUpdates }}
        </span>
      </KunButton>
    </div>

    <div v-if="tagCloud?.items.length" class="flex flex-wrap gap-2">
      <button
        v-for="t in tagCloud.items"
        :key="t.tag"
        type="button"
        :aria-pressed="tag === t.tag"
        :class="
          cn(
            'rounded-md px-2.5 py-1 text-sm transition-colors',
            tag === t.tag
              ? 'bg-primary text-white'
              : 'bg-default-100 text-default-600 hover:bg-primary/15 hover:text-primary'
          )
        "
        @click="toggleTag(t.tag)"
      >
        #{{ t.tag }}
        <span class="opacity-70">{{ t.collection_count }}</span>
      </button>
    </div>

    <div v-if="workId" class="text-default-600 flex items-center gap-2 text-sm">
      <span>只显示收录了</span>
      <KunLink :to="`/galgame/${workId}`" underline="hover">这部作品</KunLink>
      <span>的收藏夹</span>
      <KunButton variant="light" size="xs" @click="work = ''">
        <KunIcon name="lucide:x" />
        清除
      </KunButton>
    </div>

    <GalgameCollectionRowCardSkeleton v-if="pending && !items.length" />

    <template v-else-if="items.length">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <GalgameCollectionRowCard
          v-for="c in items"
          :key="c.id"
          :collection="c"
        />
      </div>
      <KunButton
        v-if="hasMore"
        variant="light"
        color="primary"
        full-width
        :loading="loadingMore"
        @click="loadMore"
      >
        <KunIcon name="lucide:chevron-down" />
        加载更多
      </KunButton>
    </template>

    <KunNull v-else-if="problem" description="收藏夹加载失败，请稍后刷新重试" />
    <KunNull v-else description="这里还没有收藏夹" />

    <KunDrawer
      v-model="followedOpen"
      placement="right"
      responsive
      size="lg"
      title="我关注的收藏夹"
    >
      <GalgameCollectionFollowedList v-if="followedMounted" />
    </KunDrawer>
  </div>
</template>
