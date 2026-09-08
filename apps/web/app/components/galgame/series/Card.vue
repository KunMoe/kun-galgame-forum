<script setup lang="ts">
const props = defineProps<{
  series: GalgameSeriesCard
}>()

const includedGamesText = computed(() => {
  if (!props.series.sample_galgame.length) {
    return '暂无 Galgame'
  }
  const names = props.series.sample_galgame.map((g) => `《${g.name}》`)
  return `${names.join('、')}${props.series.galgame_count > 5 ? ' 等' : ''}`
})

// galgame_count is what this site lists; catalog_galgame_count is the whole
// series, and it is what the detail page one click away renders. Printing only
// the first under 共 made the browse list say 「共 1 部」 about a series whose own
// page then listed three.
const countText = computed(() => {
  const listed = props.series.galgame_count
  const total = props.series.catalog_galgame_count
  if (!total || total <= listed) {
    return `共 ${listed} 部 Galgame`
  }
  return `共 ${total} 部 · 本站收录 ${listed} 部`
})
</script>

<template>
  <KunCard
    :href="`/galgame/series/${series.id}`"
    class-name="group relative flex h-full flex-col overflow-hidden backdrop-blur-none"
    :is-transparent="false"
  >
    <GalgameSeriesBanner
      :is-n-s-f-w="series.is_nsfw"
      :galgames="series.sample_galgame"
    />

    <div class="flex flex-grow flex-col px-1 pb-1">
      <h3
        class="group-hover:text-primary mb-2 line-clamp-2 text-xl font-bold transition-colors"
      >
        {{ `${series.name} 系列` }}
      </h3>

      <p class="text-default-500 mb-4 line-clamp-1 text-xs">
        {{ includedGamesText }}
      </p>

      <div class="mt-auto flex items-center justify-between">
        <div class="text-default-500 flex items-center gap-2 text-sm">
          <KunIcon name="lucide:gamepad-2" class="h-4 w-4" />
          <span>{{ countText }}</span>
        </div>

        <div
          class="group-hover:text-primary flex items-center gap-1.5 transition-all group-hover:translate-x-1"
        >
          <span class="text-sm font-medium">查看详情</span>
          <KunIcon name="lucide:arrow-right" class="h-4 w-4" />
        </div>
      </div>
    </div>
  </KunCard>
</template>
