<script setup lang="ts">
import type { EntityCardModel } from '~/utils/entityCardModel'

defineProps<{
  card: EntityCardModel
  accent: string
  badgeTone: Record<string, string>
}>()
</script>

<template>
  <span
    v-if="card.media.kind === 'image' && card.media.shape === 'logo'"
    class="kun-entity-logo flex h-5 shrink-0 items-center rounded bg-white px-0.5"
  >
    <KunImage
      :src="card.media.image.url"
      alt=""
      loading="lazy"
      object-fit="contain"
      class-name="h-full"
    />
  </span>
  <KunIcon v-else :name="card.icon" :class="cn('size-4 shrink-0', accent)" />
  <span class="text-foreground truncate font-medium">{{ card.title }}</span>
  <span class="text-default-500 shrink-0 text-xs">
    {{ [...card.chips, card.info].filter(Boolean).join(' · ') }}
  </span>
  <span
    v-for="badge in card.badges"
    :key="badge.label"
    :class="
      cn(
        'shrink-0 rounded px-1 text-[10px] leading-4 font-medium',
        badgeTone[badge.tone]
      )
    "
  >
    {{ badge.label }}
  </span>
</template>

<style scoped>
.kun-entity-logo :deep(img) {
  width: auto;
  height: 100%;
}
</style>
