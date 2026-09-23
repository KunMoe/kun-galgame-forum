<script setup lang="ts">
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'
import type { GalgameTag, WorkPage } from '#shared/utils/api/schemas'

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

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const { data: tag } = await useApi<GalgameTag>(
  () => `tag:${tag_id.value}:${allowsNsfw.value}`,
  (api) =>
    api.GET('/tags/{tag_id}', {
      params: {
        path: { tag_id: String(tag_id.value) },
        query: { include_nsfw: allowsNsfw.value }
      }
    })
)

if (!tag.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 标签',
    fatal: true
  })
}

const { page, limit, query } = useEntityWorksQuery()
const { data: works, status } = await useApi<WorkPage>(
  () => `tag-works:${tag_id.value}:${JSON.stringify(query.value)}`,
  (api) =>
    api.GET('/tags/{tag_id}/works', {
      params: { path: { tag_id: String(tag_id.value) }, query: query.value }
    })
)
const galgames = useWorkCards(() => works.value?.items)
const total = computed(() => works.value?.total ?? 0)

const data = computed(() => {
  const t = tag.value!
  return {
    name: catalogVocabularyName(t),
    category: t.is_sexual ? 'sexual' : t.tag_kind,
    hidden: t.is_hidden,
    description: pickCatalogIntro(t.intros)?.value ?? ''
  }
})

const isIndexable = computed(
  () => data.value.category !== 'sexual' && !data.value.hidden
)

if (isIndexable.value) {
  useKunSeoMeta({
    title: `标签 ${data.value.name} 的 Galgame`,
    description:
      data.value.description ||
      `含有标签「${data.value.name}」的 Galgame 作品合集, 例如 ${galgames.value
        .slice(0, 5)
        .map((g) => g.name)
        .join('、')} 等。`,
    ...(galgames.value[0]?.effective_banner_url
      ? { ogImage: galgames.value[0].effective_banner_url }
      : {})
  })
} else {
  useKunDisableSeo(`标签 ${data.value.name} 的 Galgame`)
}
</script>

<template>
  <div class="flex flex-col gap-6">
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

    <GalgameCardNav :is-show-advanced="false" axes />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该标签含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <KunLoading :loading="status === 'pending'">
      <GalgameCard
        :is-transparent="false"
        v-if="galgames.length"
        :galgames="galgames"
      />

      <KunNull v-else :description="`${data.name} 标签下暂无 Galgame`" />
    </KunLoading>

    <KunPagination
      v-if="total > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
