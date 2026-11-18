<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

useKunDisableSeo('网站设置')

const tabs = [
  { value: 'runtime', textValue: '运行期设置' },
  { value: 'env', textValue: '环境变量与密钥' },
  { value: 'changes', textValue: '配置中心变更' }
]
const activeTab = ref('runtime')
</script>

<template>
  <div class="w-full space-y-6">
    <KunHeader name="网站设置" description="本论坛的全局配置项" />

    <KunTab
      v-model="activeTab"
      :items="tabs"
      variant="underlined"
      color="primary"
      size="sm"
    />

    <template v-if="activeTab === 'runtime'">
      <AdminSettingTopicAI />
      <AdminSettingAppRelease />
      <AdminSettingTrust />
      <AdminSettingNotification />
      <AdminSettingDlsite />
    </template>
    <AdminSettingEnvVariables v-else-if="activeTab === 'env'" />
    <AdminSettingEnvChangeLog v-else />
  </div>
</template>
