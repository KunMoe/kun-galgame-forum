<script setup lang="ts">
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

const {
  page,
  limit,
  type,
  language,
  platform,
  gameType,
  sortField,
  sortOrder
} = useGalgameFilters()

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const { data, status } = await useKunFetch<GalgameSeriesDetail>(
  `/galgame-series/${series_id.value}`,
  {
    method: 'GET',
    query: {
      page,
      limit,
      type,
      language,
      platform,
      gameType,
      sortField,
      sortOrder,
      series_id
    }
  }
)

if (!data.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 系列',
    fatal: true
  })
}

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
        v-if="data.galgame.length"
        :galgames="data.galgame"
      />

      <KunNull v-else :description="`${data.name} 系列下暂无 Galgame`" />
    </KunLoading>

    <KunPagination
      v-if="data.galgame_count > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgame_count / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
