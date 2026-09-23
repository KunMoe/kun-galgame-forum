<script setup lang="ts">
import type { Series } from '#shared/utils/api/schemas'
import { seriesCardOf } from '~/utils/galgame/entityCards'

const props = defineProps<{
  series: GalgameDetailSeriesRef[]
}>()

const nameOf = useCatalogName()

// A work sits in one or two series, so each card is its own read.
const { data: cards } = useApi<Series[]>(
  () => `series-panel:${props.series.map((s) => s.id).join(',')}`,
  async (api) => {
    const found = await Promise.all(
      props.series.map((s) =>
        api.GET('/series/{series_id}', {
          params: { path: { series_id: String(s.id) } }
        })
      )
    )
    return {
      data: found.flatMap((res) => (res.data ? [res.data] : [])),
      response: new Response(null, { status: 200 })
    }
  },
  { lazy: true }
)

const data = computed(() =>
  cards.value
    ? {
        series: cards.value
          .map((s) => seriesCardOf(s, nameOf))
          .filter((s) => s.galgame_count > 0)
      }
    : undefined
)
</script>

<template>
  <div v-if="data?.series.length" class="space-y-3">
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
