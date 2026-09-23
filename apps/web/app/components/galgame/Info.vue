<script setup lang="ts">
import {
  KUN_GALGAME_AGE_LIMIT_MAP,
  getGalgameOriginalLanguageName
} from '~/constants/galgame'
import type {
  Engine,
  SeriesSummary,
  Work,
  WorkCompany
} from '#shared/utils/api/schemas'

defineProps<{
  companies: WorkCompany[]
  engines: Engine[]
  series: SeriesSummary[]
  originalLanguage: string | null
  contentRating: 'all_ages' | 'r18'
  releaseDate?: string | null
  releaseDatePrecision?: 'day' | 'month' | 'year' | null
}>()

const galgame = inject<Work>('galgame')
const nameOf = useCatalogName()

const getLanguageName = getGalgameOriginalLanguageName

const ageKey = (rating: 'all_ages' | 'r18') =>
  rating === 'r18' ? 'r18' : 'all'

const releaseText = (
  date?: string | null,
  precision?: 'day' | 'month' | 'year' | null
) => {
  if (!date) {
    return '未定 (TBA)'
  }
  if (precision === 'year') {
    return date.slice(0, 4)
  }
  if (precision === 'month') {
    return date.slice(0, 7)
  }
  return getReleaseDateText(date, false)
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    class-name="overflow-visible"
    content-class="space-y-3"
  >
    <dl class="space-y-5">
      <GalgameDetailOfficial :companies="companies" />

      <div v-if="series && series.length > 0">
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          所属系列
        </dt>
        <dd
          class="text-default-900 mt-1.5 flex flex-wrap items-center gap-x-3 text-base font-medium dark:text-white"
        >
          <KunLink
            v-for="item in series"
            :key="item.id"
            :to="`/galgame/series/${item.id}`"
            underline="none"
            class-name="text-foreground hover:text-primary text-base font-semibold"
          >
            {{ nameOf(item).name }}
          </KunLink>
        </dd>
      </div>

      <div
        v-if="engines && engines.length > 0"
        class="border-default-200 dark:border-default-700/50"
      >
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          游戏引擎
        </dt>
        <dd
          class="text-default-900 mt-1.5 flex flex-wrap items-center gap-x-3 text-base font-medium dark:text-white"
        >
          <KunLink
            v-for="item in engines"
            :key="item.id"
            :to="`/galgame/engine/${item.id}`"
            underline="none"
            class-name="text-foreground hover:text-primary text-base font-semibold"
          >
            {{ nameOf(item).name }}
            <KunTooltip
              v-if="item.catalog_work_count > 0"
              :text="`${item.catalog_work_count} 个 Galgame 使用此引擎制作`"
            >
              <KunChip size="xs">
                {{ `+ ${item.catalog_work_count}` }}
              </KunChip>
            </KunTooltip>
          </KunLink>
        </dd>
      </div>

      <div class="flex items-center justify-between">
        <dt class="text-default-500 text-sm font-medium">游戏原语言</dt>
        <dd>
          <KunChip v-if="originalLanguage" color="warning">
            {{ getLanguageName(originalLanguage) }}
          </KunChip>
          <span v-else class="text-default-500 text-sm">未公布</span>
        </dd>
      </div>

      <div class="flex items-center justify-between">
        <dt class="text-default-500 text-sm font-medium">发售日期</dt>
        <dd class="text-default-700 text-sm">
          {{ releaseText(releaseDate, releaseDatePrecision) }}
        </dd>
      </div>

      <div class="flex items-center justify-between">
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          年龄限制
        </dt>
        <dd>
          <KunTooltip
            position="left"
            :text="KUN_GALGAME_AGE_LIMIT_MAP[ageKey(contentRating)]"
          >
            <KunChip
              variant="flat"
              :color="contentRating === 'all_ages' ? 'success' : 'danger'"
            >
              {{ contentRating === 'all_ages' ? '全年龄' : 'R18' }}
            </KunChip>
          </KunTooltip>
        </dd>
      </div>
    </dl>

    <KunButton
      v-if="galgame"
      variant="flat"
      color="primary"
      size="sm"
      full-width
      @click="navigateTo(`/galgame/${galgame.id}/history`)"
    >
      <KunIcon name="lucide:history" />
      修订历史
    </KunButton>
  </KunCard>
</template>
