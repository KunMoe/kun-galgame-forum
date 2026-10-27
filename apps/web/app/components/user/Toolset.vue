<script setup lang="ts">
import type {
  PageListToolsetSummary,
  UserToolsetRelation
} from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const props = withDefaults(
  defineProps<{
    userId: number
    relation?: UserToolsetRelation
  }>(),
  { relation: 'authored' }
)

const pageData = reactive({
  page: usePageQuery(),
  limit: 24
})

const { data, status, problem, pending } = useKunApi<PageListToolsetSummary>(
  () => `user-toolsets:${props.userId}:${props.relation}:${pageData.page}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/toolsets', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: props.relation,
          page: pageData.page,
          limit: pageData.limit
        }
      },
      signal
    })
)

const emptyDescription = computed(() =>
  props.relation === 'favorited' ? '这只笨蛋萝莉还没有收藏任何工具' : '暂无工具'
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

    <UserListSkeleton v-else-if="pending" />
    <KunNull
      v-else-if="data && !data.items.length"
      :description="emptyDescription"
    />
  </div>
</template>
