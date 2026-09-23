<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type { SeriesPage } from '#shared/utils/api/schemas'
import { seriesCardOf } from '~/utils/galgame/entityCards'

const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })
const limit = 12

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)
const nameOf = useCatalogName()

const { data: seriesPage, status } = await useApi<SeriesPage>(
  () => `series:${page.value}:${allowsNsfw.value}`,
  (api) =>
    api.GET('/series', {
      params: {
        query: { page: page.value, limit, include_nsfw: allowsNsfw.value }
      }
    })
)

const data = computed(() =>
  seriesPage.value
    ? {
        series: seriesPage.value.items.map((s) => seriesCardOf(s, nameOf)),
        total: seriesPage.value.total
      }
    : undefined
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="Galgame 系列"
      description="Galgame 全系列所有 Galgame 作品。例如美少女万华镜 1, 2, 3, 4, 5, 雪女, 外传 就是一个 Galgame 系列。某个会社制作的所有 Galgame 并不算系列, 请到 Galgame 会社页面中查看"
    >
      <template #endContent>
        <span class="text-default-600 text-sm">
          {{ `总计 ${data?.total || 0} 个系列` }}
        </span>
      </template>
    </KunHeader>

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="仅显示全年龄系列"
      description="当前为 SFW 模式, 只列出全部作品均为全年龄的系列。本站收录的 Galgame 系列绝大多数含有 R18 作品, 如需查看, 请在设置面板开启 NSFW 开关。"
    />

    <div v-if="data" class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <GalgameSeriesCard
        v-for="(series, index) in data.series"
        :key="series.id"
        :style="{ animationDelay: `${index * 50}ms` }"
        :series="series"
      />
    </div>

    <KunPagination
      v-if="data && data.total > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
