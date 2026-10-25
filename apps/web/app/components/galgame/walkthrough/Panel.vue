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
  <div class="space-y-3">
    <p class="text-default-500 text-sm">
      按角色整理的选项流程: 选一个角色, 看要怎么选、在哪里存档、会到达哪个结局。
    </p>
    <KunLink
      v-for="w in walkthroughs"
      :key="w.id"
      :to="`/galgame/${workId}/walkthrough/${w.id}`"
      underline="none"
      class-name="block"
    >
      <KunCard
        :is-hoverable="true"
        :is-transparent="false"
        content-class="space-y-2"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-foreground font-medium">
            {{
              KUN_GALGAME_WALKTHROUGH_SOURCE[w.source_site]?.name ??
              w.source_site
            }}
            的攻略流程
          </span>
          <span class="text-default-500 text-xs">
            {{ w.ending_count }} 个结局 · {{ w.save_count }} 个存档点
          </span>
        </div>
        <div class="flex flex-wrap gap-1.5">
          <KunChip v-for="c in w.chart_characters" :key="c.key" size="xs">
            {{ characterName(c) }}
          </KunChip>
        </div>
      </KunCard>
    </KunLink>
  </div>
</template>
