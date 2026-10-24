<script setup lang="ts">
definePageMeta({
  middleware: 'permission',
  permissions: ['topic.manage_sections']
})

useKunDisableSeo('话题分区管理')
</script>

<template>
  <div class="w-full space-y-6">
    <KunHeader
      name="话题分区管理"
      description="调整分区与分组的显示名称、说明、图标、颜色与顺序, 或归档分区。分区的萌萌点规则与所属分组不能在这里修改。"
    />
    <AdminSectionManager />
  </div>
</template>
