<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import type {
  GalgameWalkthroughGraphEdge,
  GalgameWalkthroughGraphLayout
} from '#shared/types/galgame-walkthrough'
import {
  KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE,
  KUN_GALGAME_WALKTHROUGH_TONES
} from '~/constants/galgameWalkthrough'

const props = defineProps<{
  layout: GalgameWalkthroughGraphLayout
  routeSegments: Set<string> | null
  isFullscreen?: boolean
}>()

const emit = defineEmits<{
  expand: []
}>()

const selected = defineModel<{ kind: 'node' | 'edge'; id: string } | null>(
  'selected',
  { required: true }
)

const viewport = ref<HTMLElement | null>(null)
const content = computed(() => ({
  width: props.layout.width,
  height: props.layout.height
}))
const focus = computed(() => {
  const start = props.layout.nodes.find((n) => n.kind === 'entry')
  return start ? { x: start.x, y: start.y + 280 } : null
})
const isWide = useMediaQuery('(min-width: 640px)')
const freeTouchPan = computed(() => !!props.isFullscreen || isWide.value)

const {
  offsetX,
  offsetY,
  scale,
  isPanning,
  transform,
  fit,
  zoomBy,
  onPointerDown,
  onPointerMove,
  onPointerUp,
  onWheel
} = useOfficialGraphViewport(viewport, content, { focus, freeTouchPan })

const toneOf = (tone: number | null) =>
  tone === null
    ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE
    : KUN_GALGAME_WALKTHROUGH_TONES[
        tone % KUN_GALGAME_WALKTHROUGH_TONES.length
      ]!

const onRoute = (edge: GalgameWalkthroughGraphEdge) =>
  !props.routeSegments || props.routeSegments.has(edge.segmentKey)

const litNodes = computed(() => {
  if (!props.routeSegments) return null
  const ids = new Set<string>()
  for (const e of props.layout.edges) {
    if (onRoute(e)) {
      ids.add(e.from)
      ids.add(e.to)
    }
  }
  return ids
})

const isEdgeSelected = (edge: GalgameWalkthroughGraphEdge) =>
  selected.value?.kind === 'edge' && selected.value.id === edge.id

const DRAG_SLOP = 4
let pressedAt = { x: 0, y: 0 }
const onSurfacePointerDown = (event: PointerEvent) => {
  pressedAt = { x: event.clientX, y: event.clientY }
  onPointerDown(event)
}
const onSurfaceClick = (event: MouseEvent) => {
  if (
    Math.hypot(event.clientX - pressedAt.x, event.clientY - pressedAt.y) <=
    DRAG_SLOP
  ) {
    selected.value = null
  }
}

const uid = useId()
</script>

<template>
  <div
    ref="viewport"
    aria-label="攻略流程图, 拖拽平移, 滚轮缩放"
    :class="
      cn(
        'border-default-200 bg-default-50 relative overflow-hidden rounded-xl border select-none',
        isFullscreen
          ? 'h-full touch-none'
          : 'h-[36rem] touch-pan-y sm:h-[44rem]',
        isPanning ? 'cursor-grabbing' : 'cursor-grab'
      )
    "
    @pointerdown="onSurfacePointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
    @click.self="onSurfaceClick"
    @wheel="onWheel"
  >
    <svg
      class="pointer-events-none absolute inset-0 size-full"
      aria-hidden="true"
    >
      <defs>
        <pattern
          :id="`kun-walkthrough-grid-${uid}`"
          width="26"
          height="26"
          patternUnits="userSpaceOnUse"
          :patternTransform="`translate(${offsetX} ${offsetY}) scale(${scale})`"
        >
          <path
            d="M 26 0 L 0 0 0 26"
            fill="none"
            stroke-width="1"
            class="stroke-default-200"
          />
        </pattern>
      </defs>
      <rect
        width="100%"
        height="100%"
        :fill="`url(#kun-walkthrough-grid-${uid})`"
      />
    </svg>

    <div class="absolute top-0 left-0 origin-top-left" :style="{ transform }">
      <svg
        :width="layout.width"
        :height="layout.height"
        :viewBox="`0 0 ${layout.width} ${layout.height}`"
        class="pointer-events-none absolute top-0 left-0 overflow-visible"
        aria-hidden="true"
      >
        <g v-for="edge in layout.edges" :key="edge.id">
          <path
            :d="edge.path"
            fill="none"
            stroke-linecap="round"
            :stroke-width="
              isEdgeSelected(edge) ? 3 : edge.steps.length ? 2 : 1.25
            "
            :stroke-dasharray="edge.segmentKey ? undefined : '4 4'"
            :class="
              cn(
                'pointer-events-none transition-[opacity,stroke-width] duration-200',
                isEdgeSelected(edge)
                  ? 'stroke-primary-500'
                  : toneOf(edge.tone).stroke,
                !onRoute(edge) && 'opacity-15'
              )
            "
          />
          <path
            d="M -9 -3.6 L 0 0 L -9 3.6 L -6.4 0 Z"
            :transform="`translate(${edge.head.x} ${edge.head.y}) rotate(${edge.head.angle})`"
            :class="
              cn(
                'pointer-events-none',
                isEdgeSelected(edge)
                  ? 'fill-primary-500'
                  : toneOf(edge.tone).fill,
                !onRoute(edge) && 'opacity-15'
              )
            "
          />
          <path
            v-if="edge.steps.length"
            :d="edge.path"
            fill="none"
            stroke="transparent"
            stroke-width="14"
            class="cursor-pointer"
            style="pointer-events: stroke"
            @pointerdown.stop
            @click.stop="selected = { kind: 'edge', id: edge.id }"
          />
          <text
            v-if="edge.steps.length && onRoute(edge)"
            :x="edge.labelX"
            :y="edge.labelY"
            text-anchor="start"
            stroke-width="4"
            paint-order="stroke"
            class="fill-default-500 stroke-default-50 pointer-events-none text-[10px]"
          >
            {{ edge.steps.length }} 步
          </text>
        </g>
      </svg>

      <GalgameWalkthroughGraphNode
        v-for="node in layout.nodes"
        :key="node.id"
        :node="node"
        :is-dimmed="!!litNodes && !litNodes.has(node.id)"
        :is-selected="selected?.kind === 'node' && selected.id === node.id"
        @pointerdown.stop
        @click="selected = { kind: 'node', id: node.id }"
      />
    </div>

    <div
      class="absolute top-2 right-2 flex flex-col gap-1"
      @pointerdown.stop
      @click.stop
    >
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="放大"
        @click="zoomBy(1.2)"
      >
        <KunIcon name="lucide:plus" />
      </KunButton>
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="缩小"
        @click="zoomBy(1 / 1.2)"
      >
        <KunIcon name="lucide:minus" />
      </KunButton>
      <KunButton
        :is-icon-only="true"
        variant="flat"
        color="default"
        size="sm"
        aria-label="适应画布"
        @click="fit(true)"
      >
        <KunIcon name="lucide:scan" />
      </KunButton>
      <KunButton
        v-if="!isFullscreen"
        :is-icon-only="true"
        variant="flat"
        color="primary"
        size="sm"
        aria-label="全屏查看"
        @click="emit('expand')"
      >
        <KunIcon name="lucide:maximize" />
      </KunButton>
    </div>

    <p
      class="text-default-400 pointer-events-none absolute bottom-2 left-3 text-xs"
    >
      滚轮缩放 · 拖拽平移 · 点击连线查看选项
    </p>
  </div>
</template>
