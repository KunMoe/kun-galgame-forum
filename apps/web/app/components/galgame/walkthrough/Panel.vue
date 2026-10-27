<script setup lang="ts">
import type { Walkthrough } from '#shared/utils/api/schemas'
import { KUN_GALGAME_WALKTHROUGH_SOURCE } from '~/constants/galgameWalkthrough'

defineProps<{
  workId: string
  walkthroughs: Walkthrough[]
}>()

const nameOf = useCatalogName()

const characterName = (c: Walkthrough['chart_characters'][number]) =>
  c.catalog_character
    ? nameOf(c.catalog_character.catalog_name).name
    : c.source_name
</script>

<template>
  <div class="divide-default-200 divide-y">
    <KunLink
      v-for="w in walkthroughs"
      :key="w.id"
      :to="`/galgame/${workId}/walkthrough/${w.id}`"
      underline="none"
      class-name="group block py-3 first:pt-0 last:pb-0"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span
          class="text-foreground group-hover:text-primary font-medium transition-colors"
        >
          {{ KUN_GALGAME_WALKTHROUGH_SOURCE[w.source_site] ?? w.source_site }}
          的攻略流程
        </span>
        <span class="text-default-500 flex items-center gap-1 text-xs">
          {{ w.ending_count }} 个结局 · {{ w.save_count }} 个存档点
          <KunIcon
            name="lucide:chevron-right"
            class-name="size-4 transition-transform group-hover:translate-x-0.5"
          />
        </span>
      </div>
      <div class="mt-2 flex flex-wrap gap-1.5">
        <KunChip v-for="c in w.chart_characters" :key="c.key" size="xs">
          {{ characterName(c) }}
        </KunChip>
      </div>
    </KunLink>
  </div>
</template>
