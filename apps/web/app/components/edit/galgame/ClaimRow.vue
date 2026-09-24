<script setup lang="ts">
import type { WorkSubmissionSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  item: WorkSubmissionSummary
  timeLabel: string
}>()

const lastAt = computed(() => props.item.last_event?.created_at ?? null)
</script>

<template>
  <div
    class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
  >
    <div class="min-w-0 flex-1 space-y-1">
      <div class="flex flex-wrap items-center gap-2">
        <h3
          class="hover:text-primary truncate text-lg font-medium transition-colors"
        >
          {{ item.display_name || '(无标题)' }}
        </h3>
        <KunChip
          size="xs"
          variant="flat"
          :color="galgameClaimStateBadge(item.state).color"
        >
          {{ galgameClaimStateBadge(item.state).label }}
        </KunChip>
      </div>

      <div
        v-if="item.first_acted_at || lastAt"
        class="text-default-500 flex flex-wrap items-center gap-2 text-sm"
      >
        <span v-if="item.first_acted_at">
          {{ timeLabel }} <KunTime :time="item.first_acted_at" />
        </span>
        <template v-if="lastAt && lastAt !== item.first_acted_at">
          <span v-if="item.first_acted_at">·</span>
          <span>最后处理 <KunTime :time="lastAt" /></span>
        </template>
      </div>

      <slot name="note" />
    </div>

    <div class="flex shrink-0 gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
