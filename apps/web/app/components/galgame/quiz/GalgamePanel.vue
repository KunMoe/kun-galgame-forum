<script setup lang="ts">
import type { PageListQuizSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const route = useRoute()
const workId = computed(() => String((route.params as { id: string }).id))

const emit = defineEmits<{
  'update:loading': [boolean]
}>()

const page = usePageQuery()
const limit = 12

const { data, status, problem, pending, refresh } =
  useKunApi<PageListQuizSummary>(
    () => `quizzes-work:${workId.value}:${page.value}:${limit}`,
    (api, { signal }) =>
      api.GET('/quizzes', {
        params: {
          query: {
            page: page.value,
            limit,
            work_id: workId.value,
            include_nsfw: true
          }
        },
        signal
      })
  )
watchEffect(() =>
  emit('update:loading', status.value === 'pending' && !pending.value)
)

const showPublish = ref(false)
const openPublish = () => {
  if (!requireLogin()) return
  showPublish.value = true
}
const onPublished = () => {
  page.value = 1
  refresh()
}

const { paginationPage, isPaginationPending } = usePaginationScroll(
  page,
  () => status.value === 'pending'
)
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between gap-2">
      <KunHeader name="本作题库" scale="h3">
        <template #description>
          <p class="text-default-500 text-sm">
            与本作相关的题目, 也欢迎为本作出题
          </p>
        </template>
      </KunHeader>
      <KunButton size="sm" @click="openPublish">
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:plus" />出题
        </span>
      </KunButton>
    </div>

    <div ref="paginationStart" class="pagination-results scroll-mt-20">
      <KunNull v-if="problem" :description="problemMessage(problem)" />
      <GalgameQuizListSkeleton v-else-if="pending" :count="4" />
      <KunLoading
        v-else-if="data && data.items.length"
        :loading="isPaginationPending"
      >
        <GalgameQuizList :quizzes="data.items" />
      </KunLoading>
      <KunNull
        v-else-if="status !== 'pending'"
        description="本作还没有题目, 快来出第一题吧"
      />
    </div>

    <KunPagination
      v-if="(data?.total || 0) > limit"
      v-model:current-page="paginationPage"
      :total-page="Math.ceil((data?.total || 0) / limit)"
      :is-loading="status === 'pending'"
    />

    <GalgameQuizPublish
      v-model="showPublish"
      :work-id="workId"
      @on-published="onPublished"
    />
  </div>
</template>
