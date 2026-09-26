<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

defineProps<{ item: KunTabItem; dot: boolean }>()
</script>

<template>
  <span class="inline-flex items-center gap-2">
    <KunIcon v-if="item.icon" :name="item.icon" class="shrink-0" />
    <span>{{ item.textValue }}</span>
    <span
      v-if="dot"
      class="bg-secondary-500 size-2 shrink-0 rounded-full"
      aria-label="有新动态"
    />
  </span>
</template>
