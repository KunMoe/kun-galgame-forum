<script setup lang="ts">
import type { GalgameWalkthroughGraphNode } from '#shared/types/galgame-walkthrough'
import {
  KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE,
  KUN_GALGAME_WALKTHROUGH_TONES
} from '~/constants/galgameWalkthrough'

const props = defineProps<{
  node: GalgameWalkthroughGraphNode
  isSelected: boolean
}>()

const tone = computed(() =>
  props.node.tone === null
    ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE
    : KUN_GALGAME_WALKTHROUGH_TONES[
        props.node.tone % KUN_GALGAME_WALKTHROUGH_TONES.length
      ]!
)

const labels = useWalkthroughLabels()
const label = computed(() => labels.node(props.node))

const icon = computed(
  () =>
    ({
      entry: 'lucide:play',
      save: 'lucide:save',
      ending: 'lucide:flag',
      collect: 'lucide:images',
      open: 'lucide:more-horizontal'
    })[props.node.kind]
)
</script>

<template>
  <button
    type="button"
    :title="label"
    :style="{
      left: `${node.x - node.width / 2}px`,
      top: `${node.y - node.height / 2}px`,
      width: `${node.width}px`,
      height: `${node.height}px`
    }"
    :class="
      cn(
        'absolute flex items-center gap-1.5 overflow-hidden border px-2 text-left text-xs transition-shadow duration-200',
        node.kind === 'ending' ? 'rounded-xl font-medium' : 'rounded-full',
        node.kind === 'entry' &&
          'bg-success-100 text-success-800 border-success-400',
        node.kind === 'save' &&
          'bg-content1 text-foreground border-default-300',
        node.kind === 'ending' && tone.chip,
        (node.kind === 'collect' || node.kind === 'open') &&
          'bg-default-100 text-default-500 border-default-200 border-dashed',
        isSelected && 'ring-primary-500 ring-2'
      )
    "
  >
    <KunIcon :name="icon" class-name="size-3.5 shrink-0" />
    <span class="truncate">{{ label }}</span>
  </button>
</template>
