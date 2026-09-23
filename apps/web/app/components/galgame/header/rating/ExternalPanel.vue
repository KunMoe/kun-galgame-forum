<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_RATING_TIER_INSUFFICIENT,
  KUN_GALGAME_RATING_TIER_SCOPE_HINT,
  kunGalgameRatingTierBadge,
  type KunGalgameExternalRatingSource
} from '~/constants/galgame-rating'
import { externalRatingHistogram } from './_stats'

const props = defineProps<{
  galgame: GalgameDetail
  source: KunGalgameExternalRatingSource
}>()

const STAT_LABELS = [
  ['average', '平均分'],
  ['stdev', '标准差'],
  ['min', '最低分'],
  ['max', '最高分']
] as const

const meta = computed(() => KUN_GALGAME_EXTERNAL_RATING_MAP[props.source])

const row = computed(() =>
  props.galgame.external_ratings?.find((r) => r.source === props.source)
)

// Every source publishes something different, and what it publishes changes:
// erogamescape and vndb both gained a histogram in 2026-08. Read the capability
// off the payload instead of hardcoding it per source, so a source that gains
// data later just starts rendering.
const buckets = computed(() =>
  row.value?.distribution?.length
    ? externalRatingHistogram(row.value.distribution, meta.value.histogram.keys)
    : null
)

const categories = computed(() =>
  meta.value.histogram.keys.map(meta.value.histogram.label)
)

// The bars are a different population from the vote_count beside them on both
// sources that gained a histogram, and neither gap is a bug: VNDB's dump drops
// private lists, and 批评空间's histogram is aggregated from a mirror that syncs
// on its own cursor. Show the histogram's own denominator rather than let a
// reader add up the bars and conclude the numbers are broken.
const barsTotal = computed(() =>
  buckets.value?.reduce((sum, count) => sum + count, 0)
)

const barsNote = computed(() => {
  if (barsTotal.value == null || barsTotal.value === row.value?.vote_count) {
    return ''
  }
  const basis = meta.value.histogram.basis
  return `分布图共 ${barsTotal.value.toLocaleString('en-US')} 人, 和上面的评分人数不是同一份统计${basis ? `: ${basis}` : ''}。`
})

const tier = computed(() =>
  kunGalgameRatingTierBadge(meta.value, row.value?.score, row.value?.vote_count)
)

const tierTooltip = computed(() =>
  tier.value
    ? `${tier.value.description}, ${KUN_GALGAME_RATING_TIER_SCOPE_HINT}`
    : `${meta.value.label}的评分人数不到 ${meta.value.tier.minVotes} 人, 分级会被少数几票带偏, 所以这里不给等级`
)

const statRows = computed(() => {
  const stats = row.value?.stats
  if (!stats) {
    return []
  }
  return STAT_LABELS.flatMap(([key, label]) => {
    const value = stats[key]
    return value == null ? [] : [{ key, label, value }]
  })
})

// dlsite intentionally has no link template: the purchase URL is assembled
// server-side so it carries the affiliate id. No URL means no link, not a
// hand-built one that drops the affiliate.
const purchaseUrl = computed(() =>
  props.source === 'dlsite' ? (props.galgame.dlsite_purchase_url ?? '') : ''
)

// A space belongs between CJK and Latin, not between two CJK runs: "在 VNDB 查看"
// but "在批评空间查看".
const spacedLabel = computed(() =>
  /^[\x20-\x7e]+$/.test(meta.value.label)
    ? ` ${meta.value.label} `
    : meta.value.label
)

const link = computed(() => {
  if (props.source === 'dlsite') {
    return ''
  }
  const externalId = props.galgame.refs?.[props.source]
  return externalId ? (meta.value.link?.(externalId) ?? '') : ''
})
</script>

<template>
  <div v-if="row" class="space-y-5">
    <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
      <div>
        <div class="flex items-center gap-2">
          <div class="flex items-baseline gap-0.5">
            <span class="text-3xl leading-none font-semibold tabular-nums">
              {{ meta.formatScore(row.score) }}
            </span>
            <span class="text-default-500 text-xl leading-none font-medium">
              {{ meta.scoreSuffix }}
            </span>
          </div>
          <KunTooltip :text="tierTooltip">
            <KunChip
              size="sm"
              variant="flat"
              :color="tier ? tier.color : 'default'"
            >
              {{ tier ? tier.label : KUN_GALGAME_RATING_TIER_INSUFFICIENT }}
            </KunChip>
          </KunTooltip>
        </div>
        <span class="text-default-500 text-xs">{{ meta.hint }}</span>
      </div>

      <div class="text-default-500 flex gap-6 text-sm">
        <span>
          评分人数
          <span class="text-foreground font-medium tabular-nums">
            {{ row.vote_count.toLocaleString('en-US') }}
          </span>
        </span>
        <span v-if="row.rank">
          站内排名
          <span class="text-foreground font-medium tabular-nums">
            #{{ row.rank }}
          </span>
        </span>
      </div>

      <KunChip v-if="purchaseUrl" color="primary" variant="flat" size="sm">
        <KunIcon name="lucide:badge-check" />
        支持正版购买
      </KunChip>
    </div>

    <div v-if="buckets" class="space-y-1">
      <h4 class="font-medium">评分分布</h4>
      <GalgameHeaderRatingDistributionChart
        :work-id="galgame.id"
        :source="source"
        :buckets="buckets"
        :categories="categories"
        :unit="meta.unit"
      />
      <p v-if="barsNote" class="text-default-400 text-xs">{{ barsNote }}</p>
    </div>

    <div v-if="statRows.length" class="space-y-1.5">
      <h4 class="font-medium">分数概况</h4>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div
          v-for="stat in statRows"
          :key="stat.key"
          class="bg-default-100 rounded-lg px-3 py-2"
        >
          <div class="text-default-500 text-xs">{{ stat.label }}</div>
          <div class="font-medium tabular-nums">
            {{ Number(stat.value.toFixed(2)) }}
          </div>
        </div>
      </div>
      <p class="text-default-400 text-xs">
        这些数字和上面的分数出自同一批投票, 但上面那个是{{ meta.aggregate }},
        和这里的平均分不是同一个数。
      </p>
    </div>

    <GalgameHeaderRatingSourceCompare :galgame="galgame" :highlight="source" />

    <KunInfo
      v-if="!buckets && !statRows.length"
      color="default"
      title="外部来源只有汇总分"
      :description="`本站从百科同步到的只有${spacedLabel}的平均分、评分人数和排名, 拿不到每一条评分具体是谁打的, 所以这里没有分布图和评分人列表。`"
    />
    <KunInfo
      v-else
      color="default"
      title="外部来源没有逐人评分"
      :description="`本站从百科同步到的是${spacedLabel}的汇总统计, 拿不到每一条评分具体是谁打的, 所以这里没有评分人列表。`"
    />

    <div v-if="purchaseUrl" class="flex flex-wrap items-center gap-3">
      <GalgameDlsitePurchase
        :purchase-url="purchaseUrl"
        :coupon-url="galgame.dlsite_coupon_url"
        :campaign-name="galgame.dlsite_campaign_name"
      />
      <span class="text-default-500 text-xs">
        本站与 DLsite 官方合作, 从这里购买正版, 分成会全部回馈给用户
      </span>
    </div>

    <KunButton
      v-if="link"
      :href="link"
      target="_blank"
      rel="noopener noreferrer"
      variant="flat"
      color="primary"
      size="sm"
    >
      <span class="flex items-center gap-1">
        <KunIcon name="lucide:external-link" />
        在{{ spacedLabel }}查看
      </span>
    </KunButton>
  </div>
</template>
