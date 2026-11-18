<script setup lang="ts">
const { data, problem, pending, form, isSaving, save } = useAdminSettingsForm({
  is_trust_check_enabled: false,
  is_trust_scan_enabled: false
})
</script>

<template>
  <AdminSettingSection
    title="内容安全"
    description="话题、回复、评论等用户文本与 NextMoe 内容安全服务的两道关"
    :problem="problem"
    :pending="pending"
    :is-ready="!!data"
    :is-saving="isSaving"
    @save="save"
  >
    <div class="space-y-3">
      <KunSwitch v-model="form.is_trust_check_enabled" label="写入前检查词表" />
      <p class="text-default-500 text-xs">
        命中拒绝词的内容不会发布; 服务 500 毫秒内没有响应时放行
      </p>
      <KunSwitch v-model="form.is_trust_scan_enabled" label="写入后送审扫描" />
      <p class="text-default-500 text-xs">发布后在后台送审, 不影响发布速度</p>
    </div>
  </AdminSettingSection>
</template>
