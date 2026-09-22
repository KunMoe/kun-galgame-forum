<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { operations } from '#shared/types/api/v1'
import type { TopicSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { useCursorList } from '~/composables/useCursorList'

type TopicSort = NonNullable<
  NonNullable<operations['listTopics']['parameters']['query']>['sort']
>

const OFFERED_SORTS = [
  'bumped_desc',
  'bumped_asc',
  'created_desc',
  'created_asc',
  'views_1d_desc',
  'views_1d_asc',
  'views_7d_desc',
  'views_7d_asc',
  'views_30d_desc',
  'views_30d_asc',
  'views_desc',
  'views_asc'
] as const satisfies readonly TopicSort[]

type OfferedSort = (typeof OFFERED_SORTS)[number]
type SortField =
  | 'bumped'
  | 'created'
  | 'views_1d'
  | 'views_7d'
  | 'views_30d'
  | 'views'
type SortOrder = 'asc' | 'desc'

const SORT_FIELD_OPTIONS: { value: SortField; label: string }[] = [
  { value: 'bumped', label: '更新时间' },
  { value: 'created', label: '创建时间' },
  { value: 'views_1d', label: '日浏览数' },
  { value: 'views_7d', label: '周浏览数' },
  { value: 'views_30d', label: '月浏览数' },
  { value: 'views', label: '总浏览数' }
]

const isOfferedSort = (value: string): value is OfferedSort =>
  (OFFERED_SORTS as readonly string[]).includes(value)

const toToken = (field: SortField, order: SortOrder): OfferedSort =>
  `${field}_${order}` as OfferedSort

const fieldOf = (token: OfferedSort): SortField =>
  token.endsWith('_asc')
    ? (token.slice(0, -4) as SortField)
    : (token.slice(0, -5) as SortField)

const sortQuery = useRouteQuery<string>('sort', 'bumped_desc', {
  mode: 'replace'
})

const offeredSort = computed<OfferedSort>(() =>
  isOfferedSort(sortQuery.value) ? sortQuery.value : 'bumped_desc'
)

const sortField = computed(() => fieldOf(offeredSort.value))
const sortOrder = computed<SortOrder>(() =>
  offeredSort.value.endsWith('_asc') ? 'asc' : 'desc'
)

// v1 takes no implicit inputs — no cookie, no header — so the resolved stance
// has to travel as an explicit query parameter on every call.
const { allowsNsfw: includeNsfw } = useContentStance()

const setSortField = (value: SortField | SortField[] | null) => {
  if (!value || Array.isArray(value)) return
  if (value === sortField.value) return
  sortQuery.value = toToken(value, sortOrder.value)
}

const setSortOrder = (value: SortOrder) => {
  if (value === sortOrder.value) return
  sortQuery.value = toToken(sortField.value, value)
}

const { items, hasMore, problem, status, loadingMore, loadMore, refresh } =
  await useCursorList<TopicSummary>(
    () => `topics:${offeredSort.value}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
    (api, cursor, { signal }) =>
      api.GET('/topics', {
        params: {
          query: {
            limit: 50,
            sort: offeredSort.value,
            include_nsfw: includeNsfw.value,
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )
</script>

<template>
  <div class="flex flex-col gap-6">
    <KunHeader
      name="话题列表"
      description="鲲 Galgame 论坛的全部话题，涵盖 Galgame 讨论、技术交流、资源求助与日常闲聊，在这里和大家一起畅所欲言。"
    />

    <div class="flex items-center justify-between gap-3">
      <KunSelect
        class-name="w-44"
        :model-value="sortField"
        :options="SORT_FIELD_OPTIONS"
        @update:model-value="setSortField"
      />

      <div class="flex shrink-0 items-center gap-1">
        <KunButton
          :is-icon-only="true"
          :variant="sortOrder === 'desc' ? 'flat' : 'light'"
          color="primary"
          size="sm"
          @click="setSortOrder('desc')"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-down" />
        </KunButton>
        <KunButton
          :is-icon-only="true"
          :variant="sortOrder === 'asc' ? 'flat' : 'light'"
          color="primary"
          size="sm"
          @click="setSortOrder('asc')"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-up" />
        </KunButton>
      </div>
    </div>

    <template v-if="problem">
      <KunNull :description="problemMessage(problem)" />
      <div class="flex justify-center">
        <KunButton variant="flat" size="sm" @click="() => refresh()">
          重试
        </KunButton>
      </div>
    </template>

    <template v-else>
      <KunLoading :loading="status === 'pending'">
        <div class="divide-default-200/60 divide-y">
          <TopicCard v-for="topic in items" :key="topic.id" :topic="topic" />
        </div>
      </KunLoading>

      <KunNull
        v-if="status !== 'pending' && !items.length"
        description="真的一滴也不剩了呜呜呜"
      />

      <div v-if="items.length" class="flex justify-center pt-4">
        <KunButton
          v-if="hasMore"
          variant="light"
          :loading="loadingMore"
          @click="loadMore"
        >
          加载更多
        </KunButton>
        <span v-else class="text-default-400 text-sm">没有更多话题了</span>
      </div>
    </template>
  </div>
</template>
