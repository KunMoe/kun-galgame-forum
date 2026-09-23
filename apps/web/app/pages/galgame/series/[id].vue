<script setup lang="ts">
import type { Series, WorkPage } from '#shared/utils/api/schemas'

const route = useRoute()
const series_id = computed(() => {
  return Number((route.params as { id: string }).id)
})

if (!Number.isInteger(series_id.value) || series_id.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 系列',
    fatal: true
  })
}

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const { data: series } = await useApi<Series>(
  () => `series:${series_id.value}`,
  (api) =>
    api.GET('/series/{series_id}', {
      params: { path: { series_id: String(series_id.value) } }
    })
)

if (!series.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 系列',
    fatal: true
  })
}

const { page, limit, query } = useEntityWorksQuery()
const { data: works, status } = await useApi<WorkPage>(
  () => `series-works:${series_id.value}:${JSON.stringify(query.value)}`,
  (api) =>
    api.GET('/series/{series_id}/works', {
      params: { path: { series_id: String(series_id.value) }, query: query.value }
    })
)
const galgames = useWorkCards(() => works.value?.items)
const total = computed(() => works.value?.total ?? 0)

const nameOf = useCatalogName()
const data = computed(() => ({
  name: nameOf(series.value!).name,
  description: pickCatalogIntro(series.value!.intros)?.value ?? ''
}))

useKunSeoMeta({
  title: `${data.value.name} 系列的 Galgame`,
  description:
    data.value.description || `${data.value.name} 系列收录的 Galgame 作品合集。`
})
</script>

<template>
  <div v-if="data" class="flex flex-col gap-6">
    <KunHeader
      :name="`${data.name} 系列的 Galgame`"
      :description="data.description"
    >
      <template #endContent>
        <p class="text-default-500">
          本页展示资料库中该系列的 Galgame, 可按类型 / 语言 / 平台 /
          排序筛选。默认仅显示 SFW 的 Galgame, 查看 NSFW Galgame
          请在设置面板打开 NSFW 开关。如果有数据错误请
          <KunLink to="/doc/contact"> 联系我们 </KunLink>。
        </p>
      </template>
    </KunHeader>

    <GalgameCardNav :is-show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该系列含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <KunLoading :loading="status === 'pending'">
      <GalgameCard
        :is-transparent="false"
        v-if="galgames.length"
        :galgames="galgames"
      />

      <KunNull v-else :description="`${data.name} 系列下暂无 Galgame`" />
    </KunLoading>

    <KunPagination
      v-if="total > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
