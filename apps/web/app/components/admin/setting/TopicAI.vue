<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { Settings } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const api = useApiClient()
const { data, problem, refresh } = await useApi(
  'admin-settings',
  (client, { signal }) => client.GET('/admin/settings', { signal })
)

const form = reactive<Omit<Settings, 'is_ai_configured'>>({
  is_topic_classify_enabled: false,
  is_topic_summary_enabled: false,
  topic_ai_model: '',
  topic_ai_timeout_ms: 10000,
  topic_summary_min_length: 500,
  topic_summary_regenerate_threshold: 300
})
watch(
  data,
  (value) => {
    if (value) {
      Object.assign(form, {
        is_topic_classify_enabled: value.is_topic_classify_enabled,
        is_topic_summary_enabled: value.is_topic_summary_enabled,
        topic_ai_model: value.topic_ai_model,
        topic_ai_timeout_ms: value.topic_ai_timeout_ms,
        topic_summary_min_length: value.topic_summary_min_length,
        topic_summary_regenerate_threshold:
          value.topic_summary_regenerate_threshold
      })
    }
  },
  { immediate: true }
)

const isSaving = ref(false)
const save = async () => {
  isSaving.value = true
  const result = await settle(
    api.PATCH('/admin/settings', {
      body: {
        ...form,
        topic_ai_timeout_ms: Number(form.topic_ai_timeout_ms),
        topic_summary_min_length: Number(form.topic_summary_min_length),
        topic_summary_regenerate_threshold: Number(
          form.topic_summary_regenerate_threshold
        )
      }
    })
  )
  isSaving.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('设置已保存, 其它服务实例会在 30 秒内生效', 'success')
  await refresh()
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-5"
  >
    <div>
      <h2 class="text-lg font-semibold">话题自动分区与摘要</h2>
      <p class="text-default-500 text-sm">
        由语言模型根据话题内容决定分区, 并为较长的话题生成摘要
      </p>
    </div>

    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <template v-else-if="data">
      <p
        v-if="!data.is_ai_configured"
        class="bg-warning-50 text-warning-700 rounded-lg p-3 text-sm"
      >
        服务器没有配置模型 (KUN_AI_BASE_URL 与 KUN_AI_TOKEN),
        下面的开关暂时不会生效
      </p>

      <div class="space-y-3">
        <KunSwitch
          v-model="form.is_topic_classify_enabled"
          label="自动判断新话题的分区"
        />
        <p class="text-default-500 text-xs">
          开启后作者不再自己选择分区,
          只能在话题页申请管理员调整。模型没有响应时会退回作者选择的分区,
          之后自动复核
        </p>
        <KunSwitch
          v-model="form.is_topic_summary_enabled"
          label="为较长的话题生成摘要"
        />
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <KunInput v-model="form.topic_ai_model" label="模型 ID" />
        <KunInput
          v-model="form.topic_ai_timeout_ms"
          type="number"
          label="发布时等待模型的时间 (毫秒)"
        />
        <KunInput
          v-model="form.topic_summary_min_length"
          type="number"
          label="生成摘要的最少正文字数"
        />
        <KunInput
          v-model="form.topic_summary_regenerate_threshold"
          type="number"
          label="重新生成摘要的改动字数 (新增 + 删除)"
        />
      </div>

      <div class="flex justify-end">
        <KunButton :loading="isSaving" @click="save">保存</KunButton>
      </div>
    </template>
  </KunCard>
</template>
