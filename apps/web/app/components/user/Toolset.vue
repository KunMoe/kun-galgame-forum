<script setup lang="ts">
import type { PageListToolsetSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const props = defineProps<{
  userId: number
}>()

const pageData = reactive({
  page: usePageQuery(),
  limit: 24
})

const { data, status, problem } = await useApi<PageListToolsetSummary>(
  () => `user-toolsets:${props.userId}:${pageData.page}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/toolsets', {
      params: {
        path: { user_id: String(props.userId) },
        query: { page: pageData.page, limit: pageData.limit }
      },
      signal
    })
)
</script>

<template>
  <div class="space-y-3">
    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <div v-else-if="data && data.items.length" class="space-y-3">
      <ToolsetCard :items="data.items" />

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull v-else-if="data && !data.items.length" description="暂无工具" />
  </div>
</template>
