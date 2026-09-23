<script setup lang="ts">
import type { PageListGalgameResource } from '#shared/utils/api/schemas'

const props = defineProps<{
  workId: string
  currentId: string
}>()

const { data } = await useApi<PageListGalgameResource>(
  () => `work-resources-recs:${props.workId}`,
  (api, { signal }) =>
    api.GET('/works/{work_id}/resources', {
      params: {
        path: { work_id: props.workId },
        query: { limit: 100 }
      },
      signal
    })
)

const recommendations = computed(() =>
  (data.value?.items ?? [])
    .filter((item) => item.id !== props.currentId)
    .slice(0, 6)
)
</script>

<template>
  <KunCard
    v-if="recommendations.length"
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-4 h-full"
  >
    <div class="flex flex-col gap-3">
      <GalgameResourceCard
        v-for="resource in recommendations"
        :key="resource.id"
        :resource="resource"
      />
    </div>
  </KunCard>
</template>
