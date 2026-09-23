<script setup lang="ts">
import type { Engine, WorkPage } from '#shared/utils/api/schemas'

const route = useRoute()
const engine_id = computed(() => {
  return Number((route.params as { id: string }).id)
})

if (!Number.isInteger(engine_id.value) || engine_id.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 引擎',
    fatal: true
  })
}

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const { data: engine } = await useApi<Engine>(
  () => `engine:${engine_id.value}`,
  (api) =>
    api.GET('/engines/{engine_id}', {
      params: { path: { engine_id: String(engine_id.value) } }
    })
)

if (!engine.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到 Galgame 引擎',
    fatal: true
  })
}

const { page, limit, query } = useEntityWorksQuery()
const { data: works, status } = await useApi<WorkPage>(
  () => `engine-works:${engine_id.value}:${JSON.stringify(query.value)}`,
  (api) =>
    api.GET('/engines/{engine_id}/works', {
      params: { path: { engine_id: String(engine_id.value) }, query: query.value }
    })
)
const galgames = useWorkCards(() => works.value?.items)
const total = computed(() => works.value?.total ?? 0)

const nameOf = useCatalogName()
const data = computed(() => ({
  name: nameOf(engine.value!).name,
  description: engine.value!.description,
  alias: engine.value!.aliases
}))

useKunSeoMeta({
  title: `${data.value.name} 引擎`,
  description: `查看所有使用 ${data.value.name} 引擎制作的 Galgame`
})
</script>

<template>
  <div v-if="data" class="flex flex-col gap-6">
    <KunHeader
      :name="`${data.name} 引擎制作的 Galgame`"
      :description="data.description"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            本页展示资料库中使用该引擎制作的 Galgame, 可按类型 / 语言 / 平台 /
            排序筛选。默认仅显示 SFW 的 Galgame, 查看 NSFW Galgame
            请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <div
            v-if="data.alias.length"
            class="text-default-500 flex flex-wrap gap-2"
          >
            别名
            <KunChip
              color="primary"
              v-for="(a, index) in data.alias"
              :key="index"
            >
              {{ a }}
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
      description="当前为 SFW 模式，该引擎含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <KunLoading :loading="status === 'pending'">
      <GalgameCard
        :is-transparent="false"
        v-if="galgames.length"
        :galgames="galgames"
      />

      <KunNull v-else :description="`${data.name} 引擎下暂无 Galgame`" />
    </KunLoading>

    <KunPagination
      v-if="total > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
