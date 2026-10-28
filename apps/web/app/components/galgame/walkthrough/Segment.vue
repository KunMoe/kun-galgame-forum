<script setup lang="ts">
import type {
  WalkthroughChart,
  WalkthroughSegment
} from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_WALKTHROUGH_ENDING_TYPE,
  KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE,
  KUN_GALGAME_WALKTHROUGH_TONES
} from '~/constants/galgameWalkthrough'

const props = withDefaults(
  defineProps<{
    segment: WalkthroughSegment
    chart: WalkthroughChart
    tones: Map<string, number>
    highlightedSave: string | null
    idPrefix?: string
  }>(),
  { idPrefix: 'wt' }
)

const emit = defineEmits<{
  jump: [saveKey: string]
}>()

const labels = useWalkthroughLabels()

const toneOf = (index: number | null | undefined) =>
  index === null || index === undefined
    ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE
    : KUN_GALGAME_WALKTHROUGH_TONES[
        index % KUN_GALGAME_WALKTHROUGH_TONES.length
      ]!

const from = computed(() => props.segment.from_node)

const startLabel = computed(() => labels.start(props.chart, props.segment))

const ending = computed(() =>
  props.segment.to_node.node_type === 'ending'
    ? props.chart.chart_endings.find(
        (e) => e.key === props.segment.to_node.node_key
      )
    : undefined
)

const segmentTone = computed(() => {
  const key = segmentCharacterKey(props.chart, props.segment)
  return toneOf(key ? props.tones.get(key) : null)
})

const endingTone = computed(() =>
  toneOf(
    ending.value?.character_keys[0]
      ? props.tones.get(ending.value.character_keys[0])
      : null
  )
)

const choices = computed(() => choiceCount(props.segment.steps))
</script>

<template>
  <section
    :id="`${idPrefix}-seg-${segment.key}`"
    :class="cn('scroll-mt-24 space-y-3 border-l-2 pl-4', segmentTone.border)"
  >
    <div class="flex flex-wrap items-center gap-2">
      <KunButton
        v-if="from.node_type === 'save' && from.node_key"
        size="sm"
        variant="flat"
        color="primary"
        @click="emit('jump', from.node_key)"
      >
        <KunIcon name="lucide:save" />
        读取「{{ startLabel }}」
      </KunButton>
      <KunChip v-else-if="from.node_type === 'entry'" color="success" size="sm">
        <KunIcon name="lucide:play" />
        从【{{ startLabel }}】开始
      </KunChip>
      <KunChip v-else color="secondary" size="sm">
        <KunIcon name="lucide:corner-down-right" />
        接在「{{ startLabel }}」之后
      </KunChip>
      <KunIcon name="lucide:arrow-right" class-name="text-default-400 size-4" />
      <span
        v-if="ending"
        :class="
          cn(
            'inline-flex items-center gap-1.5 rounded-lg border px-2 py-0.5 text-sm',
            endingTone.chip
          )
        "
      >
        <KunIcon name="lucide:flag" class-name="size-3.5" />
        <GalgameWalkthroughStepText
          :text="ending.ending_name"
          :localized="ending.localized"
          size="sm"
        />
      </span>
      <span v-else class="text-default-500 text-sm">
        {{
          segment.to_node.node_type === 'collect'
            ? '回收 CG / 场景'
            : '本段结束'
        }}
      </span>
      <span class="text-default-400 text-xs">{{ choices }} 个选项</span>
    </div>

    <KunInfo v-if="segment.condition" color="warning" title="前置条件">
      <GalgameWalkthroughStepText
        :text="segment.condition"
        :localized="segment.condition_localized"
        size="sm"
      />
      <div
        v-if="requiredEndingKeys(segment).length"
        class="mt-1 flex flex-wrap items-center gap-1.5 text-xs"
      >
        <span>需先看过</span>
        <KunChip
          v-for="key in requiredEndingKeys(segment)"
          :key="key"
          size="xs"
          color="warning"
        >
          {{ labels.ending(chart, key) }}
        </KunChip>
      </div>
    </KunInfo>

    <GalgameWalkthroughStepList
      :steps="segment.steps"
      :chart="chart"
      :id-prefix="idPrefix"
      :highlighted-save="highlightedSave"
    />

    <div>
      <div
        v-if="ending"
        :class="
          cn(
            'inline-flex flex-wrap items-center gap-2 rounded-lg border px-3 py-1.5',
            endingTone.chip
          )
        "
      >
        <KunIcon name="lucide:flag" />
        <GalgameWalkthroughStepText
          :text="ending.ending_name"
          :localized="ending.localized"
          size="sm"
        />
        <span class="text-xs opacity-80">
          {{ KUN_GALGAME_WALKTHROUGH_ENDING_TYPE[ending.ending_type] }}
        </span>
      </div>
      <span
        v-else-if="segment.to_node.node_type === 'collect'"
        class="text-default-500 text-sm"
      >
        回收 CG / 场景后结束
      </span>
      <span v-else class="text-default-400 text-sm">本段到此为止</span>
    </div>
  </section>
</template>
