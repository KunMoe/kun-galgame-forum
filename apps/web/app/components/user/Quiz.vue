<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { useRouteQuery } from '@vueuse/router'
import type { PageListQuizSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const props = defineProps<{
  userId: number
}>()

const { id } = storeToRefs(usePersistUserStore())
const isOwner = computed(() => !!id.value && id.value === props.userId)
const { allowsNsfw } = useContentStance()

const tabQuery = useRouteQuery<string>('tab', 'publish', { mode: 'replace' })
const tab = computed(() =>
  tabQuery.value === 'answered' && isOwner.value ? 'answered' : 'publish'
)
const tabItems = computed<KunTabItem[]>(() => {
  const items: KunTabItem[] = [
    { value: 'publish', textValue: '出题', icon: 'lucide:pencil-line' }
  ]
  if (isOwner.value) {
    items.push({ value: 'answered', textValue: '答题', icon: 'lucide:history' })
  }
  return items
})

const page = usePageQuery()
const limit = 50
const authorId = computed(() => String(props.userId))

const { data, status, problem } = await useApi<PageListQuizSummary>(
  () =>
    tab.value === 'answered'
      ? `me-answered-quizzes:${page.value}:${limit}`
      : `quizzes-author:${authorId.value}:${page.value}:${limit}:${allowsNsfw.value}`,
  (api, { signal }) => {
    if (tab.value === 'answered') {
      return api.GET('/me/answered-quizzes', {
        params: { query: { page: page.value, limit } },
        signal
      })
    }
    return api.GET('/quizzes', {
      params: {
        query: {
          page: page.value,
          limit,
          author_id: authorId.value,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    })
  }
)

const onTab = (v: string) => {
  page.value = 1
  tabQuery.value = v
}
</script>

<template>
  <div class="space-y-3">
    <KunTab
      v-if="tabItems.length > 1"
      :model-value="tab"
      :items="tabItems"
      variant="light"
      color="primary"
      @update:model-value="onTab"
    />

    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <template v-else-if="data && data.items.length">
      <GalgameQuizList :quizzes="data.items" />
      <KunPagination
        v-if="data.total > limit"
        v-model:current-page="page"
        :total-page="Math.ceil(data.total / limit)"
        :is-loading="status === 'pending'"
      />
    </template>

    <KunNull
      v-else-if="status !== 'pending'"
      :description="tab === 'answered' ? '还没有作答记录' : '还没有出过题'"
    />
  </div>
</template>
