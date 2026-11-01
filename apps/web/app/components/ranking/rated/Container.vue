<script setup lang="ts">
import type { RatedWorkRanking, RatingSite } from '#shared/utils/api/schemas'
import type { KunTabItem } from '@kungal/ui-vue'
import {
  RATED_RANKING_LIMIT,
  RATING_SITES,
  RATING_SITE_META,
  ratedRankingPageMeta
} from '~/constants/ratedRanking'

const props = defineProps<{ site: RatingSite | null }>()

const route = useRoute()
const { page, sort, query } = useRatedRankingQuery(() => props.site)

const { data, status, problem, pending } = useKunApi<RatedWorkRanking>(
  () => `rated-ranking:${JSON.stringify(query.value)}`,
  (api, { signal }) =>
    api.GET('/rankings/rated-works', {
      params: { query: query.value },
      signal
    })
)
watch(problem, (p) => {
  if (p) {
    reportProblem(p)
  }
})

const entries = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const totalPage = computed(() =>
  Math.min(
    Math.ceil(total.value / RATED_RANKING_LIMIT),
    Math.floor(10000 / RATED_RANKING_LIMIT)
  )
)

const siteHref = (target: RatingSite | null) => {
  const params = new URLSearchParams()
  for (const [key, raw] of Object.entries(route.query)) {
    const value = Array.isArray(raw) ? raw[0] : raw
    if (!value || key === 'page' || (key === 'minSites' && target)) {
      continue
    }
    if (
      key === 'sort' &&
      !ratedRankingPreset(target, value as typeof sort.value)
    ) {
      continue
    }
    params.set(key, value)
  }
  const path = target ? `/ranking/${target}` : '/ranking'
  const search = params.toString()
  return search ? `${path}?${search}` : path
}

const siteTabs = computed<KunTabItem[]>(() => [
  {
    value: 'consensus',
    textValue: '综合',
    icon: 'lucide:layers',
    href: siteHref(null)
  },
  ...RATING_SITES.map((site) => ({
    value: site,
    textValue: RATING_SITE_META[site].label,
    href: siteHref(site)
  }))
])

const meta = computed(() => ratedRankingPageMeta(props.site))
const scoreLabel = computed(() =>
  props.site ? `${RATING_SITE_META[props.site].short} 原分` : '综合评分'
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader :name="meta.title" :description="meta.description">
      <template #endContent>
        <div class="space-y-3">
          <KunTab
            :model-value="site ?? 'consensus'"
            :items="siteTabs"
            variant="underlined"
            color="primary"
          />
          <RankingRatedNav
            :site="site"
            :sort="sort"
            :total="data ? total : null"
            :pending="status === 'pending'"
            :fitted-at="data?.model_fitted_at"
            :facets="data?.facets"
          />
        </div>
      </template>
    </KunHeader>

    <RankingListSkeleton v-if="pending" height="5.25rem" />

    <KunNull
      v-else-if="problem && !data"
      description="评分排行暂时不可用, 请稍后再试"
    />

    <KunLoading v-else :loading="status === 'pending'">
      <section v-if="entries.length">
        <div
          class="border-default-200 text-default-400 flex items-center gap-3 border-b px-2 pb-2 text-xs sm:gap-4 sm:px-3"
        >
          <span class="w-7 shrink-0 text-center sm:w-9">排名</span>
          <span class="flex-1">作品</span>
          <span class="w-16 text-right sm:w-28">{{ scoreLabel }}</span>
          <span class="w-8 shrink-0" />
        </div>
        <ol class="divide-default-200 divide-y">
          <li v-for="entry in entries" :key="entry.work_summary.id">
            <RankingRatedEntry :entry="entry" :site="site" :sort="sort" />
          </li>
        </ol>
      </section>
      <GalgameOriginalLanguageNull
        v-else
        description="没有找到符合条件的 Galgame"
      />
    </KunLoading>

    <KunPagination
      v-if="totalPage > 1"
      v-model:current-page="page"
      :total-page="totalPage"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
