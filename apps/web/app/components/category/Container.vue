<script setup lang="ts">
import type { components } from '#shared/types/api/v1'
import { KUN_TOPIC_SECTION } from '~/constants/topic'

defineProps<{
  sections: components['schemas']['Section'][]
  categoryName: string
}>()
</script>

<template>
  <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
    <KunCard
      :is-transparent="false"
      v-for="section in sections"
      :key="section.section"
      :href="`/section/${section.section}`"
    >
      <div class="mb-4 flex items-center justify-between">
        <h2 class="text-lg font-semibold">
          {{ KUN_TOPIC_SECTION[section.section] }}
        </h2>
        <div class="text-default-500 flex items-center gap-4 text-sm">
          <div class="flex items-center gap-2 text-inherit">
            <KunIcon name="lucide:newspaper" />
            {{ formatNumber(section.topic_count) }}
          </div>
          <div class="flex items-center gap-2 text-inherit">
            <KunIcon name="lucide:eye" />
            {{ formatNumber(section.view_count) }}
          </div>
        </div>
      </div>

      <div class="space-y-2">
        <h3 class="line-clamp-1">
          {{ section.latest_topic?.title }}
        </h3>
        <p class="text-default-500 text-sm">
          <KunTime :time="section.latest_topic?.created_at || ''" />
        </p>
      </div>
    </KunCard>
  </div>
</template>
