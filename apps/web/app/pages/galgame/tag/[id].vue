<script setup lang="ts">
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'

const route = useRoute()
const tag_id = computed(() => {
  return Number((route.params as { id: string }).id)
})

if (!Number.isInteger(tag_id.value) || tag_id.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 标签',
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

const { data, status } = await useKunFetch<GalgameTagDetail>(
  `/galgame-tag/${tag_id.value}`,
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
      tag_id
    }
  }
)

if (!data.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 标签',
    fatal: true
  })
}

const isIndexable = computed(
  () => data.value?.category !== 'sexual' && !data.value?.hidden
)

if (isIndexable.value) {
  useKunSeoMeta({
    title: `标签 ${data.value.name} 的 Galgame`,
    description:
      data.value.description ||
      `含有标签「${data.value.name}」的 Galgame 作品合集, 例如 ${data.value.galgame
        .slice(0, 5)
        .map((g) => g.name)
        .join('、')} 等。`,
    ...(data.value.galgame[0]?.effective_banner_url
      ? { ogImage: data.value.galgame[0].effective_banner_url }
      : {})
  })
} else {
  useKunDisableSeo(`标签 ${data.value.name} 的 Galgame`)
}
</script>

<template>
  <div v-if="data" class="flex flex-col gap-6">
    <KunHeader
      :name="`含有标签 ${data.name} 的 Galgame`"
      :description="data.description"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            本页展示资料库中含有该标签的 Galgame, 可按类型 / 语言 / 平台 /
            排序筛选。默认仅显示 SFW 的 Galgame, 查看 NSFW Galgame
            请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <div class="text-default-500">
            标签类别
            <KunChip
              :color="
                data.category === 'content'
                  ? 'primary'
                  : data.category === 'sexual'
                    ? 'danger'
                    : 'success'
              "
            >
              {{ KUN_GALGAME_TAG_CATEGORY_MAP[data.category] }}
            </KunChip>
          </div>
        </div>
      </template>
    </KunHeader>

    <GalgameCardNav :is-show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该标签含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <KunLoading :loading="status === 'pending'">
      <GalgameCard
        :is-transparent="false"
        v-if="data.galgame.length"
        :galgames="data.galgame"
      />

      <KunNull v-else :description="`${data.name} 标签下暂无 Galgame`" />
    </KunLoading>

    <KunPagination
      v-if="data.galgame_count > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgame_count / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
