<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

const routeOfficialId = Number((useRoute().params as { id: string }).id)

const { page, limit, query } = useEntityWorksQuery()
const { state: viaState, set: setViaQuery } = useQueryState(
  {
    page: queryPage(),
    via: queryEnum(['', 'own', 'imprint'] as const, '')
  },
  { pageKey: 'page' }
)
const via = computed(() => viaState.value.via)

const { allowsNsfw } = useContentStance()
const isSfwMode = computed(() => !allowsNsfw.value)

const { graph, layout, hasRelations, relatedCount } =
  useCompanyRelations(routeOfficialId)

const [
  { officialId, data, pending },
  { galgames, total, status, own, imprint, pending: worksPending }
] = await Promise.all([
  useGalgameOfficialDetail(),
  useCompanyWorks(
    routeOfficialId,
    () => ({ ...query.value, ...(via.value ? { via: via.value } : {}) }),
    () => ({
      page: 1,
      limit: 1,
      sort: 'resource_updated_desc',
      include_nsfw: query.value.include_nsfw,
      include_all_original_languages: query.value.include_all_original_languages
    })
  )
])

const viaTabs = computed<KunTabItem[]>(() => [
  { value: '', textValue: `全部 ${own.value + imprint.value}` },
  { value: 'own', textValue: `自有 ${own.value}` },
  { value: 'imprint', textValue: `经旗下厂牌 ${imprint.value}` }
])
const isViaShown = computed(() => imprint.value > 0 || via.value !== '')
const setVia = (value: string | number) =>
  setViaQuery({ via: value as '' | 'own' | 'imprint' })

const worksDescription = computed((previous?: string) => {
  if (status.value === 'pending') {
    return previous ?? ''
  }
  const expected =
    via.value === 'own'
      ? own.value
      : via.value === 'imprint'
        ? imprint.value
        : own.value + imprint.value
  return total.value === expected ? '' : `当前筛选条件下共 ${total.value} 部`
})

useKunSeoMeta(() => {
  const official = data.value
  return official
    ? {
        title: `${official.name} 会社`,
        description: `${official.name}${official.alias?.length ? `, 即 ${official.alias.join('| ')}` : ''}, 查看会社 ${official.name} 制作的所有 Galgame`,
        ogCard: { kind: 'official', id: official.id }
      }
    : {}
})

const listStart = useTemplateRef('listStart')
const revealList = useRevealListStart(
  listStart,
  page,
  () => status.value === 'pending'
)
</script>

<template>
  <div>
    <div v-if="data" class="space-y-6">
      <GalgameOfficialHero
        :company="data"
        :work-count="own + imprint"
        :related-count="relatedCount"
      />

      <div
        :class="
          cn(
            'grid grid-cols-1 items-start gap-6',
            hasRelations && 'lg:grid-cols-[20rem_minmax(0,1fr)]'
          )
        "
      >
        <aside
          v-if="hasRelations && graph && layout"
          class="order-last lg:sticky lg:top-20 lg:order-none lg:max-h-[calc(100dvh-6rem)] lg:overflow-y-auto"
        >
          <GalgameOfficialRelationGraph
            :official-id="officialId"
            :graph="graph"
            :layout="layout"
            :related-count="relatedCount"
          />
        </aside>

        <section class="flex min-w-0 flex-col gap-4">
          <KunHeader
            ref="listStart"
            class="scroll-mt-20"
            :name="`${data.name}制作的全部 Galgame 作品`"
            :description="worksDescription"
            scale="h2"
          >
            <template #title><KunPageLabel :page="page" /></template>
          </KunHeader>

          <div v-if="isViaShown" class="space-y-1">
            <KunTab
              :items="viaTabs"
              :model-value="via"
              variant="pills"
              color="primary"
              size="sm"
              @update:model-value="setVia"
            />
            <p class="text-default-400 text-xs">
              自有: 由 {{ data.name }} 直接制作发行; 经旗下厂牌: 由
              {{ data.name }} 旗下的品牌制作发行。
            </p>
          </div>

          <GalgameCardNav :is-show-advanced="false" />

          <KunInfo
            v-if="isSfwMode"
            color="warning"
            title="部分 Galgame 已隐藏"
            description="当前为 SFW 模式，该会社含 NSFW 内容的 Galgame 不会显示，也不计入数量。如需查看，请在设置面板开启 NSFW 开关。"
          />

          <GalgameCardGridSkeleton v-if="worksPending" />

          <KunLoading v-else :loading="status === 'pending'">
            <GalgameCard
              v-if="galgames.length"
              :is-transparent="false"
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

            <GalgameOriginalLanguageNull
              v-else
              :description="`${data.name} 会社下暂无 Galgame`"
            />
          </KunLoading>

          <KunPagination
            v-if="total > limit"
            v-model:current-page="page"
            :total-page="pageCountOf(total, limit)"
            :is-loading="status === 'pending'"
            @update:current-page="revealList"
          />

          <p class="text-default-400 text-xs">
            资料来自 鲲 Galgame 论坛。如果有数据错误请
            <KunLink to="/doc/contact" size="sm">联系我们</KunLink>。
          </p>
        </section>
      </div>
    </div>
    <GalgameEntityDetailSkeleton v-else-if="pending" />
  </div>
</template>
