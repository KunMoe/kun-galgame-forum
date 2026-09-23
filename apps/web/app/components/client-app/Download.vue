<script setup lang="ts">
import {
  KUN_APP_DOWNLOAD_PAGE,
  KUN_APP_PLATFORMS
} from '~/constants/app-release'

const { data, problem: error } = await useApi(
  'app-version',
  (api, { signal }) => api.GET('/app/version', { signal })
)

const platforms = computed(() =>
  KUN_APP_PLATFORMS.map((platform) => {
    const url = data.value?.downloads[platform.key] ?? ''
    return {
      ...platform,
      url,
      available: !!url && url.replace(/\/$/, '') !== KUN_APP_DOWNLOAD_PAGE
    }
  })
)
</script>

<template>
  <div class="min-h-[calc(100dvh-6rem)] space-y-6">
    <KunHeader
      :name="`${kungal.name} App`"
      :description="`在手机和电脑上浏览话题、Galgame 资料与资源, 登录后即可回复与互动。App 由 ${kungal.name} 官方直接发布, 请只从本页下载。`"
    />

    <KunNull v-if="error" description="暂时无法获取 App 版本信息，请稍后再试" />

    <template v-else-if="data">
      <KunCard :is-hoverable="false" content-class="space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="text-xl">最新版本</h2>
          <KunChip color="primary" variant="flat">
            v{{ data.latest_version }}
          </KunChip>
        </div>
        <p
          v-if="data.notes"
          class="text-default-600 text-sm whitespace-pre-line"
        >
          {{ data.notes }}
        </p>
      </KunCard>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <KunCard
          v-for="platform in platforms"
          :key="platform.key"
          :is-hoverable="false"
          content-class="flex h-full flex-col gap-3"
        >
          <div class="flex items-center gap-2">
            <KunIcon :name="platform.icon" class="size-6" />
            <h3 class="text-lg font-medium">{{ platform.label }}</h3>
          </div>
          <p class="text-default-500 flex-1 text-sm">{{ platform.hint }}</p>
          <KunButton
            v-if="platform.available"
            color="primary"
            full-width
            icon
            :href="platform.url"
            target="_blank"
            rel="noopener"
          >
            <template #icon>
              <KunIcon name="lucide:download" />
            </template>
            下载 {{ platform.label }} 版
          </KunButton>
          <KunButton v-else variant="flat" full-width disabled>
            即将推出
          </KunButton>
        </KunCard>
      </div>
    </template>
  </div>
</template>
