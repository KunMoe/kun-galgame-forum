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

const shown = computed(() =>
  walkthroughText(props.text, props.localized, mode.value)
)
</script>

<template>
  <span class="inline-flex min-w-0 flex-col">
    <span
      :class="
        cn(
          'text-foreground break-words',
          size === 'sm' ? 'text-sm' : 'text-base'
        )
      "
      :lang="mode === 'zh' && !shown.sub ? 'zh-Hans' : 'ja'"
    >
      {{ shown.main }}
    </span>
    <span
      v-if="shown.sub"
      class="text-default-500 text-xs break-words"
      lang="zh-Hans"
    >
      {{ shown.sub }}
    </span>
  </span>
</template>
