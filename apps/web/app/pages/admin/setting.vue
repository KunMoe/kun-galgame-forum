<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

useKunDisableSeo('网站设置')
</script>

<template>
  <div class="w-full space-y-6">
    <KunHeader name="网站设置" description="本论坛的全局配置项" />

    <AdminSettingTopicAI />
  </div>
</template>
