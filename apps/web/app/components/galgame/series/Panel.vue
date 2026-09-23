<script setup lang="ts">
import type { SeriesSummary } from '#shared/utils/api/schemas'
import { seriesCardOf } from '~/utils/galgame/entityCards'

const props = defineProps<{
  series: SeriesSummary[]
}>()

const nameOf = useCatalogName()

const data = computed(() => ({
  series: props.series
    .map((s) => seriesCardOf(s, nameOf))
    .filter((s) => s.galgame_count > 0)
}))
</script>

<template>
  <div v-if="data.series.length" class="space-y-3">
    <KunHeader
      name="Galgame 系列"
      description="Galgame 全系列所有 Galgame 作品。例如美少女万华镜 1, 2, 3, 4, 5, 雪女, 外传 就是一个 Galgame 系列"
      scale="h3"
    />

    <div class="grid grid-cols-1 gap-6">
      <GalgameSeriesCard
        v-for="item in data.series"
        :key="item.id"
        :series="item"
      />
    </div>
  </div>
</template>
