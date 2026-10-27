<script setup lang="ts">
import type { KunWalkthroughTextMode } from '~/constants/galgameWalkthrough'

const props = defineProps<{
  text: string
  localized?: Record<string, { value: string; is_machine: boolean }>
  size?: 'sm' | 'md'
}>()

const mode = inject<Ref<KunWalkthroughTextMode>>(
  'walkthroughTextMode',
  ref('both')
)
const textLang = inject<Ref<string>>('walkthroughTextLang', ref('ja'))

const shown = computed(() =>
  walkthroughText(props.text, props.localized, mode.value, textLang.value)
)
</script>

<template>
  <span
    :class="
      cn(
        'text-foreground min-w-0 break-words',
        size === 'sm' ? 'text-sm' : 'text-base'
      )
    "
  >
    <span :lang="shown.lang">{{ shown.main }}</span>
    <span v-if="shown.sub" class="text-default-500" :lang="textLang">
      ({{ shown.sub }})</span
    >
  </span>
</template>
