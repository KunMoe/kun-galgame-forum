<script setup lang="ts">
const { data, problem, pending, form, isSaving, save } = useAdminSettingsForm({
  notification_retention_days: 365
})
</script>

<template>
  <AdminSettingSection
    title="通知保留"
    description="每天 03:45 删除超过保留天数的已读通知, 未读通知不会被删除"
    :problem="problem"
    :pending="pending"
    :is-ready="!!data"
    :is-saving="isSaving"
    @save="save"
  >
    <KunInput
      v-model="form.notification_retention_days"
      type="number"
      label="已读通知保留天数 (0 为永久保留)"
    />
  </AdminSettingSection>
</template>
