<script setup lang="ts">
import type {
  RatedWorkRankingEntry,
  RatedWorkRankingSort,
  RatingSite
} from '#shared/utils/api/schemas'
import { RATING_SITE_META, ratingSiteLabel } from '~/constants/ratedRanking'
import {
  formatDeviation,
  formatSiteScore,
  formatTopShare,
  formatVotes
} from './format'

const props = defineProps<{
  entry: RatedWorkRankingEntry
  site: RatingSite | null
  sort: RatedWorkRankingSort
}>()

const MEDALS = [
  'fluent-emoji-flat:1st-place-medal',
  'fluent-emoji-flat:2nd-place-medal',
  'fluent-emoji-flat:3rd-place-medal'
]

const nameOf = useCatalogName()
const { isBlurred } = useContentStance()

const work = computed(() => props.entry.work_summary)
const names = computed(() => nameOf(work.value))
const href = computed(() => `/galgame/${work.value.id}`)

const medal = computed(() =>
  props.sort === 'upper_asc' ? undefined : MEDALS[props.entry.rank - 1]
)

const meta = computed(() => {
  const { consensus, site_ratings } = props.entry
  const rating = props.site
    ? site_ratings.find((r) => r.site === props.site)
    : undefined
  const stats =
    rating && props.site
      ? [
          `${formatVotes(rating.vote_count)} 票`,
          `${RATING_SITE_META[props.site].short} ${formatTopShare(rating.percentile)}`
        ]
      : consensus
        ? [
            `${consensus.site_count} 站`,
            `${formatVotes(consensus.vote_count)} 票`,
            `综合${formatTopShare(consensus.percentile)}`
          ]
        : []
  return [
    names.value.original !== names.value.name ? names.value.original : '',
    work.value.maker ? nameOf(work.value.maker).name : '',
    work.value.release_date?.slice(0, 4) ?? '',
    ...stats
  ].filter(Boolean)
})

const hasResource = computed(
  () => work.value.is_published && work.value.resource_platforms.length > 0
)

const isDetailOpen = ref(false)
</script>

<template>
  <article class="hover:bg-default-100/60 px-2 py-3 transition-colors sm:px-3">
    <div
      class="grid grid-cols-[1.75rem_3rem_minmax(0,1fr)_auto_auto] items-center gap-x-3 gap-y-2 sm:grid-cols-[2.25rem_3rem_minmax(0,1fr)_auto_auto] sm:gap-x-4 sm:gap-y-1.5"
    >
      <div class="flex justify-center sm:row-span-2">
        <Icon v-if="medal" :name="medal" class="size-7" />
        <span
          v-else
          class="text-default-500 text-base font-semibold tabular-nums"
        >
          {{ entry.rank }}
        </span>
      </div>

      <KunLink
        :to="href"
        underline="none"
        class-name="block overflow-hidden rounded-md sm:row-span-2"
      >
        <GalgameCardCover
          :src="work.cover?.url"
          :alt="names.name"
          :thumbhash="work.cover?.thumbhash ?? undefined"
          :masked="isBlurred && work.is_nsfw"
        />
      </KunLink>

      <div class="min-w-0 space-y-1 sm:self-end">
        <div class="flex min-w-0 items-center gap-2">
          <KunLink
            :to="href"
            color="default"
            underline="hover"
            class-name="line-clamp-2 font-semibold sm:line-clamp-1"
          >
            {{ names.name }}
          </KunLink>
          <KunChip
            v-if="hasResource"
            size="xs"
            variant="flat"
            color="success"
            class-name="hidden shrink-0 sm:inline-flex"
          >
            本站有资源
          </KunChip>
        </div>
        <p class="text-default-500 truncate text-xs tabular-nums">
          {{ meta.join(' · ') }}
        </p>
      </div>

      <RankingRatedScore
        :entry="entry"
        :site="site"
        :sort="sort"
        class="sm:row-span-2"
      />

      <KunButton
        variant="light"
        size="sm"
        :is-icon-only="true"
        class-name="sm:row-span-2"
        :aria-label="isDetailOpen ? '收起各站详情' : '展开各站详情'"
        :aria-expanded="isDetailOpen"
        @click="isDetailOpen = !isDetailOpen"
      >
        <KunIcon
          name="lucide:chevron-down"
          :class="
            cn('size-4 transition-transform', isDetailOpen && 'rotate-180')
          "
        />
      </KunButton>

      <div
        class="col-span-4 col-start-2 flex flex-wrap gap-1 self-start sm:col-span-1 sm:col-start-3"
      >
        <KunChip
          v-if="hasResource"
          size="xs"
          variant="flat"
          color="success"
          class-name="sm:hidden"
        >
          本站有资源
        </KunChip>
        <KunChip
          v-if="site && entry.consensus"
          size="xs"
          variant="flat"
          color="secondary"
        >
          <span class="opacity-70">综合</span>
          <span class="font-semibold tabular-nums">
            {{ formatDeviation(entry.consensus.deviation) }}
          </span>
        </KunChip>
        <KunChip
          v-for="r in entry.site_ratings"
          :key="r.site"
          size="xs"
          variant="flat"
          :color="r.site === site ? 'primary' : 'default'"
        >
          <span class="opacity-70">{{ ratingSiteLabel(r.site) }}</span>
          <span class="font-semibold tabular-nums">
            {{ formatSiteScore(r.published_score, r.scale_max) }}
          </span>
        </KunChip>
      </div>
    </div>

    <RankingRatedDetails
      v-if="isDetailOpen"
      :ratings="entry.site_ratings"
      class="mt-3 sm:ml-29"
    />
  </article>
</template>
