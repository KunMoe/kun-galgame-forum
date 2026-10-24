<script setup lang="ts">
definePageMeta({
  middleware: 'permission',
  permissions: ['topic.edit_any']
})

useKunDisableSeo('分区调整申请')

const tabs = [
  { value: 'pending', textValue: '待处理' },
  { value: 'approved', textValue: '已同意' },
  { value: 'rejected', textValue: '已拒绝' }
]
const activeTab = useTabQuery('pending')
const state = computed(
  () => activeTab.value as 'pending' | 'approved' | 'rejected'
)
</script>

<template>
  <div class="w-full space-y-6">
    <KunHeader
      name="分区调整申请"
      description="作者申请移动话题, 或者自动分区复核认为话题放错了分区。同意后话题会被移动, 并按新旧分区补扣或退还萌萌点; 无论同意与否都会通知作者。"
    />
    <KunTab
      v-model="activeTab"
      :items="tabs"
      variant="underlined"
      color="primary"
      size="sm"
    />
    <AdminSectionRequestList :state="state" />
  </div>
</template>
