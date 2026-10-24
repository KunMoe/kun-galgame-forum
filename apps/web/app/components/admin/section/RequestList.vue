<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { TopicSectionRequest } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  state: TopicSectionRequest['state']
}>()

const api = useApiClient()
const { sectionLabel } = useTopicSections()
const deciding = ref('')

const { data, problem, refresh } = await useApi(
  () => `admin-section-requests:${props.state}`,
  (client, { signal }) =>
    client.GET('/admin/section-requests', {
      params: { query: { state: props.state } },
      signal
    })
)
const items = computed(() => data.value?.items ?? [])

const decide = async (
  request: TopicSectionRequest,
  state: 'approved' | 'rejected'
) => {
  deciding.value = request.id
  const result = await settle(
    api.PATCH('/admin/section-requests/{section_request_id}', {
      params: { path: { section_request_id: request.id } },
      body: { state }
    })
  )
  deciding.value = ''
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(state === 'approved' ? '已移动话题' : '已拒绝申请', 'success')
  await refresh()
}
</script>

<template>
  <div class="space-y-3">
    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <KunNull v-else-if="!items.length" description="没有申请" />

    <KunCard
      v-for="request in items"
      :key="request.id"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-2"
    >
      <div class="flex flex-wrap items-center gap-2">
        <KunLink :to="`/topic/${request.topic_id}`" class-name="font-medium">
          {{ request.topic_title || `话题 ${request.topic_id}` }}
        </KunLink>
        <span class="text-default-500 text-sm">
          {{ sectionLabel(request.from_section) }}
          <KunIcon name="lucide:arrow-right" class="mx-1 inline size-3" />
          {{ sectionLabel(request.to_section) }}
        </span>
      </div>
      <p class="text-sm">{{ request.reason }}</p>
      <div class="text-default-500 flex flex-wrap items-center gap-3 text-xs">
        <span v-if="request.requester" class="flex items-center gap-1">
          <KunAvatar :user="toKunUser(request.requester)" size="xs" />
          {{ request.requester.name ?? '已注销用户' }}
        </span>
        <span v-else>自动分区复核</span>
        <KunTime :time="request.created_at" />
        <span v-if="request.decided_at">
          处理于 <KunTime :time="request.decided_at" />
        </span>
      </div>
      <div v-if="request.state === 'pending'" class="flex justify-end gap-2">
        <KunButton
          size="sm"
          variant="light"
          color="danger"
          :disabled="!!deciding"
          @click="decide(request, 'rejected')"
        >
          拒绝
        </KunButton>
        <KunButton
          size="sm"
          color="primary"
          :loading="deciding === request.id"
          :disabled="!!deciding"
          @click="decide(request, 'approved')"
        >
          同意并移动
        </KunButton>
      </div>
    </KunCard>
  </div>
</template>
