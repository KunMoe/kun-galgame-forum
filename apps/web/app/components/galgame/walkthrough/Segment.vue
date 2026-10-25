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

const props = defineProps<{
  segment: WalkthroughSegment
  chart: WalkthroughChart
  tones: Map<string, number>
  highlightedSave: string | null
}>()

const emit = defineEmits<{
  jump: [saveKey: string]
}>()

const toneOf = (index: number | null | undefined) =>
  index === null || index === undefined
    ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE
    : KUN_GALGAME_WALKTHROUGH_TONES[
        index % KUN_GALGAME_WALKTHROUGH_TONES.length
      ]!

const from = computed(() => props.segment.from_node)

const startLabel = computed(() => nodeLabel(props.chart, from.value))

const ending = computed(() =>
  props.segment.to_node.node_type === 'ending'
    ? props.chart.chart_endings.find(
        (e) => e.key === props.segment.to_node.node_key
      )
    : undefined
)

const endingTone = computed(() =>
  toneOf(
    ending.value?.character_keys[0]
      ? props.tones.get(ending.value.character_keys[0])
      : null
  )
)

const saveLabel = (key: string | null | undefined) =>
  props.chart.save_points.find((s) => s.key === key)?.label ?? '存档'

const numbered = computed(() => {
  let choiceNo = 0
  return props.segment.steps.map((step) => ({
    step,
    no: step.step_type === 'choice' ? ++choiceNo : 0
  }))
})
</script>

<template>
  <KunCard
    :id="`wt-seg-${segment.key}`"
    :is-hoverable="false"
    :is-transparent="false"
    class-name="scroll-mt-24"
    content-class="space-y-3"
  >
    <div class="flex flex-wrap items-center gap-2">
      <template v-if="from.node_type === 'save' && from.node_key">
        <KunButton
          size="sm"
          variant="flat"
          color="primary"
          @click="emit('jump', from.node_key)"
        >
          <KunIcon name="lucide:save" />
          读取「{{ startLabel }}」
        </KunButton>
        <span class="text-default-400 text-xs">点击跳到这个存档的建立位置</span>
      </template>
      <KunChip v-else-if="from.node_type === 'entry'" color="success" size="sm">
        <KunIcon name="lucide:play" />
        从【{{ startLabel }}】开始
      </KunChip>
      <KunChip v-else color="secondary" size="sm">
        <KunIcon name="lucide:corner-down-right" />
        接在「{{ startLabel }}」之后
      </KunChip>
    </div>

    <KunInfo v-if="segment.condition" color="warning" title="前置条件">
      <GalgameWalkthroughStepText
        :text="segment.condition"
        :localized="segment.condition_localized"
        size="sm"
      />
    </KunInfo>

    <ol class="space-y-2">
      <li
        v-for="({ step, no }, i) in numbered"
        :key="i"
        :id="step.save_key ? `wt-save-${step.save_key}` : undefined"
        :class="
          cn(
            'flex scroll-mt-24 items-start gap-3 rounded-lg transition-colors duration-500',
            step.save_key &&
              highlightedSave === step.save_key &&
              'bg-primary-100'
          )
        "
      >
        <template v-if="step.step_type === 'save'">
          <span
            class="bg-primary-500 mt-1 flex size-6 shrink-0 items-center justify-center rounded-full text-white"
          >
            <KunIcon name="lucide:save" class-name="size-3.5" />
          </span>
          <span class="text-primary-700 py-0.5 text-sm font-medium">
            在这里存档:「{{ saveLabel(step.save_key) }}」
          </span>
        </template>
        <template v-else>
          <span
            :class="
              cn(
                'mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-medium',
                step.step_type === 'choice'
                  ? 'bg-default-100 text-default-700'
                  : 'bg-warning-100 text-warning-700'
              )
            "
          >
            <template v-if="step.step_type === 'choice'">{{ no }}</template>
            <KunIcon v-else name="lucide:gamepad-2" class-name="size-3.5" />
          </span>
          <div class="min-w-0 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <GalgameWalkthroughStepText
                v-if="step.step_text"
                :text="step.step_text"
                :localized="step.localized"
              />
              <KunChip
                v-if="step.is_achievement_only"
                size="xs"
                color="warning"
              >
                成就
              </KunChip>
            </div>
            <div
              v-if="step.alternatives.length"
              class="text-default-500 flex flex-wrap items-center gap-x-2 text-xs"
            >
              <span>差分 CG 回收时改选</span>
              <GalgameWalkthroughStepText
                v-for="(alt, j) in step.alternatives"
                :key="j"
                :text="alt.text"
                :localized="alt.localized"
                size="sm"
              />
            </div>
          </div>
        </template>
      </li>
    </ol>

    <div class="border-default-200 border-t pt-3">
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
  </KunCard>
</template>
