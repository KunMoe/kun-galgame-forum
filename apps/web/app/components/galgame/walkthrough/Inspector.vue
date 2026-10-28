<script setup lang="ts">
import type { WalkthroughChart } from '#shared/utils/api/schemas'
import type { GalgameWalkthroughGraphLayout } from '#shared/types/galgame-walkthrough'
import { KUN_GALGAME_WALKTHROUGH_ENDING_TYPE } from '~/constants/galgameWalkthrough'

const props = defineProps<{
  layout: GalgameWalkthroughGraphLayout
  chart: WalkthroughChart
}>()

const selected = defineModel<{ kind: 'node' | 'edge'; id: string } | null>(
  'selected',
  { required: true }
)

const emit = defineEmits<{
  openSegment: [segmentKey: string]
}>()

const labels = useWalkthroughLabels()

const edge = computed(() =>
  selected.value?.kind === 'edge'
    ? props.layout.edges.find((e) => e.id === selected.value!.id)
    : undefined
)
const node = computed(() =>
  selected.value?.kind === 'node'
    ? props.layout.nodes.find((n) => n.id === selected.value!.id)
    : undefined
)
const nodeName = (id: string) => {
  const n = props.layout.nodes.find((x) => x.id === id)
  return n ? labels.node(n) : ''
}

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
    ? props.layout.edges.filter(
        (e) => e.to === node.value!.id && e.steps.length
      )
    : []
)
</script>

<template>
  <div class="lg:border-default-200 space-y-3 lg:border-l lg:pl-4">
    <template v-if="edge">
      <p class="text-default-500 text-xs">
        {{ nodeName(edge.from) }} → {{ nodeName(edge.to) }}
      </p>
      <GalgameWalkthroughStepList
        :steps="edge.steps"
        :chart="chart"
        size="sm"
      />
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
        <p class="text-foreground font-medium">{{ labels.node(node) }}</p>
        <KunChip v-if="ending" size="xs" color="secondary">
          {{ KUN_GALGAME_WALKTHROUGH_ENDING_TYPE[ending.ending_type] }}
        </KunChip>
      </div>
      <section v-for="e in arriving" :key="e.id" class="space-y-2">
        <p class="text-default-500 text-xs">
          从「{{ nodeName(e.from) }}」到这里要选
        </p>
        <GalgameWalkthroughStepList :steps="e.steps" :chart="chart" size="sm" />
      </section>
      <div v-if="leaving.length" class="space-y-1.5">
        <p class="text-default-500 text-xs">从这里继续</p>
        <div class="flex flex-wrap gap-2">
          <KunButton
            v-for="e in leaving"
            :key="e.id"
            size="sm"
            variant="flat"
            color="default"
            @click="selected = { kind: 'edge', id: e.id }"
          >
            → {{ nodeName(e.to) }}
            <span v-if="choiceCount(e.steps)" class="text-default-500 text-xs">
              {{ choiceCount(e.steps) }} 个选项
            </span>
          </KunButton>
        </div>
      </div>
    </template>

    <p v-else class="text-default-500 text-sm">
      点击连线上的「N 个选项」查看这一段要选的选项; 点击存档或结局,
      查看到达它要选的选项和接下来的去向。
    </p>
  </div>
</template>
