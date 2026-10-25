<script setup lang="ts">
import type { WalkthroughChart } from '#shared/utils/api/schemas'
import type { GalgameWalkthroughGraphLayout } from '#shared/types/galgame-walkthrough'
import { KUN_GALGAME_WALKTHROUGH_ENDING_TYPE } from '~/constants/galgameWalkthrough'

const props = defineProps<{
  layout: GalgameWalkthroughGraphLayout
  chart: WalkthroughChart
  selected: { kind: 'node' | 'edge'; id: string } | null
}>()

const emit = defineEmits<{
  openSegment: [segmentKey: string]
}>()

const edge = computed(() =>
  props.selected?.kind === 'edge'
    ? props.layout.edges.find((e) => e.id === props.selected!.id)
    : undefined
)
const node = computed(() =>
  props.selected?.kind === 'node'
    ? props.layout.nodes.find((n) => n.id === props.selected!.id)
    : undefined
)
const nodeById = (id: string) => props.layout.nodes.find((n) => n.id === id)

const ending = computed(() =>
  node.value?.kind === 'ending'
    ? props.chart.chart_endings.find((e) => e.key === node.value!.key)
    : undefined
)

const leaving = computed(() =>
  node.value
    ? props.layout.edges.filter(
        (e) => e.from === node.value!.id && e.segmentKey
      )
    : []
)
const arriving = computed(() =>
  node.value
    ? props.layout.edges.filter((e) => e.to === node.value!.id && e.segmentKey)
    : []
)
</script>

<template>
  <div class="lg:border-default-200 space-y-3 lg:border-l lg:pl-4">
    <template v-if="edge">
      <p class="text-default-500 text-xs">
        {{ nodeById(edge.from)?.label }} → {{ nodeById(edge.to)?.label }}
      </p>
      <ol class="space-y-2">
        <li v-for="(step, i) in edge.steps" :key="i" class="flex gap-2">
          <span class="text-default-400 w-5 shrink-0 text-right text-xs">{{
            i + 1
          }}</span>
          <GalgameWalkthroughStepText
            v-if="step.step_text"
            :text="step.step_text"
            :localized="step.localized"
            size="sm"
          />
        </li>
      </ol>
      <KunButton
        size="sm"
        variant="flat"
        color="primary"
        @click="emit('openSegment', edge.segmentKey)"
      >
        <KunIcon name="lucide:list" />
        在路线列表中查看
      </KunButton>
    </template>

    <template v-else-if="node">
      <div class="space-y-1">
        <p class="text-foreground font-medium">{{ node.label }}</p>
        <template v-if="ending">
          <GalgameWalkthroughStepText
            :text="ending.ending_name"
            :localized="ending.localized"
            size="sm"
          />
          <KunChip size="xs" color="secondary">
            {{ KUN_GALGAME_WALKTHROUGH_ENDING_TYPE[ending.ending_type] }}
          </KunChip>
        </template>
      </div>
      <p class="text-default-500 text-sm">
        {{ arriving.length }} 条路线到达 · {{ leaving.length }} 条路线从这里出发
      </p>
      <div v-if="leaving.length" class="flex flex-wrap gap-2">
        <KunButton
          v-for="e in leaving"
          :key="e.id"
          size="sm"
          variant="light"
          color="default"
          @click="emit('openSegment', e.segmentKey)"
        >
          → {{ nodeById(e.to)?.label }}
        </KunButton>
      </div>
    </template>

    <p v-else class="text-default-500 text-sm">
      点击一条连线, 查看这一段要选的选项; 点击存档或结局, 查看它连接的路线。
    </p>
  </div>
</template>
