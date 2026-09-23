<script setup lang="ts">
const { page, limit, query } = useEntityWorksQuery()

const { officialId, data } = await useGalgameOfficialDetail('/game')

const { galgames, total, status, own, imprint } = await useCompanyWorks(
  officialId,
  () => query.value
)

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const official = data.value
if (official) {
  useKunSeoMeta({
    title: `${official.name} 制作的 Galgame`,
    description: `浏览会社 ${official.name} 制作的全部 Galgame, 可按类型 / 语言 / 平台 / 作品类型筛选与排序。`
  })
}
</script>

<template>
  <div v-if="data" class="flex flex-col gap-6">
    <KunHeader :name="`${data.name} 制作的 Galgame`">
      <template v-if="data.logo" #headerEndContent>
        <GalgameOfficialBrandMark
          :src="data.logo"
          :name="data.name"
          size="md"
        />
      </template>

      <template v-if="imprint" #endContent>
        <div class="flex flex-wrap items-center gap-2">
          <KunChip color="primary">自有 {{ own }}</KunChip>
          <KunChip color="secondary">
            经旗下 {{ imprint }}
          </KunChip>
        </div>
      </template>
    </KunHeader>

    <GalgameOfficialDetailNav
      :official-id="officialId"
      :galgame-count="total"
    />

    <GalgameCardNav :is-show-advanced="false" axes />

    <KunInfo
      v-if="isSfwMode"
      color="warning"
      title="部分 Galgame 已隐藏"
      description="当前为 SFW 模式，该会社含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
    />

    <KunLoading :loading="status === 'pending'">
      <GalgameCard
        :is-transparent="false"
        v-if="galgames.length"
        :galgames="galgames"
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
      v-if="total > limit"
      v-model:current-page="page"
      :total-page="Math.ceil(total / limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
