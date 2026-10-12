<script setup lang="ts">
import type { TagSummary } from '#shared/utils/api/schemas'

defineProps<{ subtags: TagSummary[]; descendantCount: number }>()
</script>

<template>
  <section v-if="subtags.length" class="space-y-2">
    <h2
      class="text-default-700 flex items-baseline gap-2 text-sm font-semibold"
    >
      子标签
      <span class="text-default-400 text-xs font-normal">
        直属 {{ subtags.length }} 个<template
          v-if="descendantCount > subtags.length"
          >, 连同更细分的共
          {{ descendantCount.toLocaleString('en-US') }} 个</template
        >
      </span>
    </h2>

    <KunScrollShadow axis="vertical" class-name="max-h-48">
      <div class="flex flex-wrap gap-1.5">
        <KunLink
          v-for="sub in subtags"
          :key="sub.id"
          underline="none"
          :to="`/galgame/tag/${sub.id}`"
        >
          <KunChip
            size="sm"
            variant="flat"
            :color="sub.is_sexual ? 'danger' : 'default'"
            class-name="cursor-pointer"
          >
            {{ catalogVocabularyName(sub) }}
            <span class="text-xs tabular-nums opacity-60">
              {{ sub.catalog_work_count.toLocaleString('en-US') }}
            </span>
            <span v-if="sub.descendant_count" class="text-xs opacity-60">
              · {{ sub.descendant_count }} 个子标签
            </span>
          </KunChip>
        </KunLink>
      </div>
    </KunScrollShadow>
  </section>
</template>
