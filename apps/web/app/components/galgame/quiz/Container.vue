<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { KunTabItem } from '@kungal/ui-vue'
import {
  quizCategoryOptions,
  quizTypeOptions,
  quizDifficultyOptions,
  quizSortFieldOptions
} from './_filters'
import { KUN_QUIZ_SORT_FIELD_CONST } from '~/constants/galgame-quiz'
import type {
  PageListQuizSummary,
  QuizCategory,
  QuizSort,
  QuizType
} from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const route = useRoute()
const userStore = usePersistUserStore()
const isLoggedIn = computed(() => !!userStore.id)
const { allowsNsfw } = useContentStance()

const opts = { mode: 'replace' as const }
const page = useRouteQuery('page', 1, { ...opts, transform: Number })
const tab = useRouteQuery<'all' | 'mine'>('tab', 'all', opts)
const category = useRouteQuery<string>('category', 'all', opts)
const type = useRouteQuery<string>('type', 'all', opts)
const difficulty = useRouteQuery('difficulty', 0, {
  ...opts,
  transform: Number
})
const sortField = useRouteQuery<string>('sort_field', 'bumped_at', opts)
const sortOrder = useRouteQuery<'asc' | 'desc'>('sort_order', 'desc', opts)
const limit = 50

const activeTab = computed(() => (isLoggedIn.value ? tab.value : 'all'))

const sort = computed<QuizSort>(() => {
  const mapped =
    sortField.value === 'update_time'
      ? 'bumped_at'
      : sortField.value === 'time'
        ? 'created'
        : sortField.value
  const field = (KUN_QUIZ_SORT_FIELD_CONST as readonly string[]).includes(
    mapped
  )
    ? mapped
    : 'bumped_at'
  const order = sortOrder.value === 'asc' ? 'asc' : 'desc'
  return `${field}_${order}` as QuizSort
})

const listQuery = computed(() => {
  const query: {
    page: number
    limit: number
    sort: QuizSort
    include_nsfw: boolean
    quiz_type?: QuizType
    quiz_category?: QuizCategory
    difficulty?: number
  } = {
    page: page.value,
    limit,
    sort: sort.value,
    include_nsfw: allowsNsfw.value
  }
  if (type.value !== 'all') {
    query.quiz_type = type.value as QuizType
  }
  if (category.value !== 'all') {
    query.quiz_category = category.value as QuizCategory
  }
  if (difficulty.value > 0) {
    query.difficulty = difficulty.value
  }
  return query
})

const { data, status, problem, refresh } = await useApi<PageListQuizSummary>(
  () =>
    activeTab.value === 'mine'
      ? `me-answered-quizzes:${page.value}:${limit}`
      : `quizzes:${JSON.stringify(listQuery.value)}`,
  (api, { signal }) => {
    if (activeTab.value === 'mine') {
      return api.GET('/me/answered-quizzes', {
        params: { query: { page: page.value, limit } },
        signal
      })
    }
    return api.GET('/quizzes', {
      params: { query: listQuery.value },
      signal
    })
  }
)

const listPath = route.path
watch(
  () => route.fullPath,
  () => {
    if (route.path !== listPath) return
    if (import.meta.client) window.scrollTo({ top: 0, behavior: 'smooth' })
  }
)

watch(
  sortField,
  (v) => {
    if (v === 'update_time') sortField.value = 'bumped_at'
    else if (v === 'time') sortField.value = 'created'
  },
  { immediate: true }
)

watch([category, type, difficulty, sortField, sortOrder, tab], () => {
  page.value = 1
})

const tabItems: KunTabItem[] = [
  { value: 'all', textValue: '全部题库', icon: 'lucide:library-big' },
  { value: 'mine', textValue: '我的答题', icon: 'lucide:history' }
]
const onTab = (v: string) => {
  tab.value = v as 'all' | 'mine'
}
const setSortOrder = (order: 'asc' | 'desc') => {
  sortOrder.value = order
}

const showPublish = ref(false)
const openPublish = () => {
  if (!requireLogin()) return
  showPublish.value = true
}
const onPublished = () => {
  tab.value = 'all'
  page.value = 1
  refresh()
}
</script>

<template>
  <div class="space-y-3">
    <div class="space-y-2">
      <KunHeader name="Galgame 题库">
        <template #description>
          <p class="text-default-500">
            由 Galgame 爱好者共同建设的题库, 支持单选、多选、判断等题型。出题合格有奖励。
          </p>
        </template>
      </KunHeader>

      <div class="flex items-center gap-2">
        <KunTab
          v-if="isLoggedIn"
          :model-value="activeTab"
          :items="tabItems"
          variant="light"
          color="primary"
          @update:model-value="onTab"
        />
        <KunButton class="ml-auto shrink-0" @click="openPublish">
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:plus" />
            出题
          </span>
        </KunButton>
      </div>

      <div
        v-if="activeTab === 'all'"
        class="flex w-full shrink-0 flex-wrap items-center justify-between gap-3 sm:flex-nowrap"
      >
        <div class="grid w-full grid-cols-2 gap-3 lg:grid-cols-4">
          <KunSelect v-model="category" :options="quizCategoryOptions" />
          <KunSelect v-model="type" :options="quizTypeOptions" />
          <KunSelect v-model="difficulty" :options="quizDifficultyOptions" />
          <KunSelect v-model="sortField" :options="quizSortFieldOptions" />
        </div>

        <div class="flex items-center gap-2">
          <KunButton
            :is-icon-only="true"
            :variant="sortOrder === 'desc' ? 'flat' : 'light'"
            size="md"
            @click="setSortOrder('desc')"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-down" />
          </KunButton>
          <KunButton
            :is-icon-only="true"
            :variant="sortOrder === 'asc' ? 'flat' : 'light'"
            size="md"
            @click="setSortOrder('asc')"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-up" />
          </KunButton>
        </div>
      </div>
    </div>

    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <GalgameQuizList
      v-else-if="data && data.items.length"
      :quizzes="data.items"
    />
    <KunNull
      v-else-if="status !== 'pending'"
      :description="
        activeTab === 'mine'
          ? '你还没有作答过任何题目'
          : '还没有人出题, 快来出第一题吧'
      "
    />

    <KunPagination
      v-if="(data?.total || 0) > limit"
      v-model:current-page="page"
      :total-page="Math.ceil((data?.total || 0) / limit)"
      :is-loading="status === 'pending'"
    />

    <GalgameQuizPublish v-model="showPublish" @on-published="onPublished" />
  </div>
</template>
