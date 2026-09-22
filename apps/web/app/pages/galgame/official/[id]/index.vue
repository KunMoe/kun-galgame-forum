<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'

const route = useRoute()

const carriedFilters = Object.fromEntries(
  Object.entries(route.query).filter(([key]) =>
    (GALGAME_FILTER_QUERY_KEYS as readonly string[]).includes(key)
  )
)
if (Object.keys(carriedFilters).length) {
  await navigateTo(
    { path: `${route.path}/game`, query: route.query },
    { replace: true }
  )
}

const GALGAME_PREVIEW_LIMIT = 8

const { officialId, data } = await useGalgameOfficialDetail({
  page: 1,
  limit: GALGAME_PREVIEW_LIMIT
})

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const gamePath = computed(
  () => `${taxonomyDetailPath('official', officialId)}/game`
)

const categoryText = (category: string) =>
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP[category] || category

const INTRO_CLAMP_CHARS = 100
const isIntroExpanded = ref(false)

const worksDescription = computed(() => {
  const detail = data.value
  if (!detail?.galgame_count) return ''
  if (!detail.imprint_galgame_count) {
    return `资料库中有 ${detail.galgame_count} 部, 下面是最近更新的几部。`
  }
  return `资料库中自有 ${detail.own_galgame_count} 部 · 经旗下厂牌 ${detail.imprint_galgame_count} 部, 下面是最近更新的几部。`
})

const official = data.value
if (official && !official.moved_to) {
  useKunSeoMeta({
    title: `${official.name} 会社`,
    description: `${official.name}${official.alias?.length ? `, 即 ${official.alias.join('| ')}` : ''}, 查看会社 ${official.name} 制作的所有 Galgame`,
    ogCard: { kind: 'official', id: official.id }
  })
}
</script>

<template>
  <div v-if="data && !data.moved_to" class="space-y-6">
    <KunHeader :name="data.name" :description="data.original">
      <template v-if="data.logo" #headerEndContent>
        <GalgameOfficialBrandMark :src="data.logo" :name="data.name" />
      </template>

      <template #endContent>
        <div class="space-y-3">
          <div class="flex flex-wrap items-center gap-2">
            <KunChip color="primary">{{ categoryText(data.category) }}</KunChip>
            <KunChip v-if="data.lang" color="secondary">
              {{ KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[data.lang] || data.lang }}
            </KunChip>

            <KunLink
              v-for="link in data.links"
              :key="link.url"
              :is-show-anchor-icon="true"
              target="_blank"
              rel="noopener noreferrer"
              underline="hover"
              size="sm"
              :to="link.url"
            >
              {{ link.name }}
            </KunLink>
          </div>

          <div
            v-if="data.alias.length"
            class="text-default-500 flex flex-wrap items-center gap-2 text-sm"
          >
            别名
            <KunChip size="xs" v-for="(a, index) in data.alias" :key="index">
              {{ a }}
            </KunChip>
          </div>

          <div v-if="data.description" class="space-y-1">
            <p
              :class="
                cn(
                  'text-default-600 whitespace-pre-line',
                  !isIntroExpanded && 'line-clamp-3'
                )
              "
            >
              {{ data.description }}
            </p>
            <p v-if="data.description_machine" class="text-default-400 text-xs">
              该简介由机器翻译生成
            </p>
            <KunButton
              v-if="data.description.length > INTRO_CLAMP_CHARS"
              variant="light"
              size="sm"
              color="primary"
              @click="isIntroExpanded = !isIntroExpanded"
            >
              {{ isIntroExpanded ? '收起简介' : '展开简介' }}
            </KunButton>
          </div>
        </div>
      </template>
    </KunHeader>

    <GalgameOfficialDetailNav
      :official-id="officialId"
      :galgame-count="data.galgame_count"
    />

    <div class="space-y-3">
      <KunHeader name="作品" :description="worksDescription" scale="h3" />

      <GalgameCard
        v-if="data.galgame.length"
        :is-transparent="false"
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

      <KunButton
        v-if="data.galgame_count > GALGAME_PREVIEW_LIMIT"
        variant="flat"
        color="primary"
        :full-width="true"
        :href="gamePath"
      >
        <KunIcon name="lucide:layout-grid" />
        浏览全部 {{ data.galgame_count }} 部作品
      </KunButton>

      <KunNull
        v-if="!data.galgame_count"
        :description="`${data.name} 会社下暂无 Galgame`"
      />
    </div>

    <GalgameOfficialRelationGraph :official-id="officialId" />

    <p class="text-default-400 text-xs">
      本页展示资料库中该会社的作品, 资料来自 NextMoe 目录。<template
        v-if="isSfwMode"
        >当前为 SFW 模式, 该会社含 NSFW 内容的 Galgame
        不计入上方数量。</template
      >如果有数据错误请
      <KunLink to="/doc/contact" size="sm">联系我们</KunLink>。
    </p>
  </div>
</template>
