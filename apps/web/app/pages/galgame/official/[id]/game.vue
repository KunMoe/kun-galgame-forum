<script setup lang="ts">
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

const { officialId, data, status } = await useGalgameOfficialDetail(
  {
    page,
    limit,
    type,
    language,
    platform,
    gameType,
    sortField,
    sortOrder
  },
  '/game'
)

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const official = data.value
if (official && !official.moved_to) {
  useKunSeoMeta({
    title: `${official.name} 制作的 Galgame`,
    description: `浏览会社 ${official.name} 制作的全部 Galgame, 可按类型 / 语言 / 平台 / 作品类型筛选与排序。`
  })
}
</script>

<template>
  <div v-if="data && !data.moved_to" class="flex flex-col gap-6">
    <KunHeader :name="`${data.name} 制作的 Galgame`">
      <template v-if="data.logo" #headerEndContent>
        <GalgameOfficialBrandMark
          :src="data.logo"
          :name="data.name"
          size="md"
        />
      </template>

      <template v-if="data.imprint_galgame_count" #endContent>
        <div class="flex flex-wrap items-center gap-2">
          <KunChip color="primary">自有 {{ data.own_galgame_count }}</KunChip>
          <KunChip color="secondary">
            经旗下 {{ data.imprint_galgame_count }}
          </KunChip>
        </div>
      </template>
    </KunHeader>

    <GalgameOfficialDetailNav
      :official-id="officialId"
      :galgame-count="data.galgame_count"
    />

    <GalgameCardNav :is-show-advanced="false" />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该会社含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <KunLoading :loading="status === 'pending'">
      <GalgameCard
        :is-transparent="false"
        v-if="data.galgame.length"
        :galgames="data.galgame"
        :hide-company="data.name"
      >
        <template #meta="{ galgame }">
          <GalgameOfficialViaImprint
            v-if="galgame.via_official"
            :name="galgame.via_official.name"
          />
        </template>
      </GalgameCard>

      <KunNull v-else :description="`${data.name} 会社下暂无 Galgame`" />
    </KunLoading>

    <KunPagination
      v-if="data.galgame_count > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(data.galgame_count / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
