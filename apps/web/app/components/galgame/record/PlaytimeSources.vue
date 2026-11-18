<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type {
  PlaytimeSource,
  WorkViewerPlaytime
} from '#shared/utils/api/schemas'

const props = defineProps<{
  workId: number
  sources: PlaytimeSource[]
}>()

const emits = defineEmits<{
  removed: [clientId: string, left: WorkViewerPlaytime]
}>()

const api = useApiClient()
const removing = ref('')

const appName = (source: PlaytimeSource) => source.app_name ?? '已删除的应用'

const remove = async (source: PlaytimeSource) => {
  const ok = await useComponentMessageStore().alert(
    `删除「${appName(source)}」记录的时长`,
    '删除后该应用的这份记录会从 NextMoe 所有站点消失, 该应用之后上报的时长会重新记录。'
  )
  if (!ok) return
  removing.value = source.client_id
  const result = await settle(
    api.DELETE('/works/{work_id}/playtime/sources/{client}', {
      params: {
        path: { work_id: String(props.workId), client: source.client_id }
      }
    })
  )
  removing.value = ''
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  emits('removed', source.client_id, result.data)
  useMessage('已删除', 'success')
}
</script>

<template>
  <div v-if="sources.length" class="space-y-2">
    <p class="text-default-500 text-xs">
      其它应用记录的时长。作品上显示的是各应用中最多的那一份,
      上面填写的只是本站的记录
    </p>
    <div
      v-for="source in sources"
      :key="source.client_id"
      class="bg-default-100 flex items-center gap-2 rounded-lg px-3 py-2 text-sm"
    >
      <span class="min-w-0 flex-1 truncate">{{ appName(source) }}</span>
      <span class="text-default-500 tabular-nums">
        {{ formatDurationMinutes(source.minutes) || '0 分钟' }}
      </span>
      <KunButton
        size="xs"
        variant="light"
        color="danger"
        :loading="removing === source.client_id"
        @click="remove(source)"
      >
        删除
      </KunButton>
    </div>
  </div>
</template>
