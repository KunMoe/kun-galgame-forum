<script setup lang="ts">
const { data, problem, pending, form, isSaving, save } = useAdminSettingsForm({
  is_topic_classify_enabled: false,
  is_topic_summary_enabled: false,
  topic_ai_model: '',
  topic_ai_timeout_ms: 10000,
  topic_summary_min_length: 500,
  topic_summary_regenerate_threshold: 300
})
</script>

<template>
  <AdminSettingSection
    title="话题自动分区与摘要"
    description="由语言模型根据话题内容决定分区, 并为较长的话题生成摘要"
    :problem="problem"
    :pending="pending"
    :is-ready="!!data"
    :is-saving="isSaving"
    @save="save"
  >
    <p
      v-if="!data?.is_ai_configured"
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
  </AdminSettingSection>
</template>
