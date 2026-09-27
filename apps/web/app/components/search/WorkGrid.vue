<script setup lang="ts">
import type { WorkSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  works: WorkSummary[]
  keywords?: string
}>()

const cards = useWorkCards(() => props.works)
</script>

<template>
  <GalgameCard :galgames="cards">
    <template #name="{ galgame }">
      <SearchHighlight :text="galgame.name" :keywords="keywords" />
    </template>
  </GalgameCard>
</template>
