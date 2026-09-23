<script setup lang="ts">
import type { PageListToolsetSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'

const { page, limit, listQuery } = useToolsetFilters()

const { data, status, problem } = await useApi<PageListToolsetSummary>(
  () => `toolsets:${JSON.stringify(listQuery.value)}`,
  (api, { signal }) =>
    api.GET('/toolsets', {
      params: { query: listQuery.value },
      signal
    })
)
</script>

<template>
  <div>
    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <div v-else-if="data" class="flex flex-col gap-3">
      <div class="z-10">
        <KunHeader
          name="Galgame 工具资源下载"
          description="Galgame 工具合集，模拟器, 翻译器, 解包工具, 补丁工具, 资源转换工具, 汉化工具, 开发者工具, 游戏管理工具, 自动化脚本 等 Galgame 工具资源下载"
        >
          <template #endContent>
            <ToolsetCardNav />
          </template>
        </KunHeader>
      </div>

      <KunLoading :loading="status === 'pending'">
        <ToolsetCard v-if="data.items.length" :items="data.items" />
        <KunNull v-else description="没有找到符合条件的工具资源" />
      </KunLoading>

      <KunCard
        v-if="data.items.length"
        :is-hoverable="false"
        :is-transparent="false"
        content-class="gap-3"
      >
        <KunPagination
          v-model:current-page="page"
          :total-page="Math.ceil(data.total / limit)"
          :is-loading="status === 'pending'"
        />
      </KunCard>
    </div>
  </div>
</template>
