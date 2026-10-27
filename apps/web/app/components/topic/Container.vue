<script setup lang="ts">
import type { operations } from '#shared/types/api/v1'
import type { TopicSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { useElementVisibility } from '@vueuse/core'
import { useCursorList } from '~/composables/useCursorList'

const props = withDefaults(defineProps<{ sectionKey?: string }>(), {
  sectionKey: ''
})

const { sectionOf } = useTopicSections()
const section = computed(() => sectionOf(props.sectionKey))
const isUnknownSection = computed(() => !!props.sectionKey && !section.value)

useKunSeoMeta(() => ({
  title: section.value?.label ?? '话题列表',
  description:
    section.value?.description ??
    `${kungal.titleShort} 话题列表, Galgame 讨论, Galgame 攻略, 长文评价, 新作资讯, 游戏运行, 开发技术, ACGN, 日常杂谈`
}))

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

const { sort: sortQuery } = useQueryState({ sort: queryString('bumped_desc') })

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
  useCursorList<TopicSummary>(
    () =>
      `topics:${props.sectionKey || 'all'}:${offeredSort.value}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
    (api, cursor, { signal }) =>
      api.GET('/topics', {
        params: {
          query: {
            limit: 50,
            sort: offeredSort.value,
            include_nsfw: includeNsfw.value,
            ...(props.sectionKey
              ? { section: props.sectionKey as TopicSummary['section'] }
              : { hidden_sections: 'exclude' as const }),
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )

const sentinel = useTemplateRef<HTMLElement>('sentinel')
const isSentinelVisible = useElementVisibility(sentinel, {
  rootMargin: '0px 0px 400px 0px'
})
watch([isSentinelVisible, hasMore, loadingMore], ([visible, more, busy]) => {
  if (visible && more && !busy) {
    loadMore()
  }
})
</script>

<template>
  <div>
    <Transition name="kun-page" mode="out-in">
      <div :key="sectionKey" class="flex flex-col gap-6">
        <TopicSectionHeader v-if="section" :section="section" />
        <KunHeader
          v-else
          name="全部话题"
          description="按更新时间排列的全部话题。技术求助与寻求资源不在这里, 可以进入对应分区查看"
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

        <KunNull v-if="isUnknownSection" description="没有这个分区" />

        <template v-else-if="problem">
          <KunNull :description="problemMessage(problem)" />
          <div class="flex justify-center">
            <KunButton variant="flat" size="sm" @click="() => refresh()">
              重试
            </KunButton>
          </div>
        </template>

        <template v-else>
          <TopicCardSkeleton v-if="status === 'pending'" />
          <template v-else>
            <div class="divide-default-200/60 divide-y">
              <TopicCard
                v-for="topic in items"
                :key="topic.id"
                :topic="topic"
              />
            </div>
          </template>

          <KunNull
            v-if="status !== 'pending' && !items.length"
            description="真的一滴也不剩了呜呜呜"
          />

          <div
            v-if="status !== 'pending' && items.length"
            ref="sentinel"
            class="text-default-400 flex justify-center pt-4 text-sm"
          >
            <span v-if="hasMore" class="flex items-center gap-2">
              <KunIcon name="svg-spinners:90-ring-with-bg" />
              正在加载更多话题
            </span>
            <span v-else>没有更多话题了</span>
          </div>
        </template>
      </div>
    </Transition>
  </div>
</template>
