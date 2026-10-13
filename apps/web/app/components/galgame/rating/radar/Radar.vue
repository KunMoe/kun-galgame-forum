<script setup lang="ts">
import {
  KUN_GALGAME_DIMENSIONS,
  KUN_GALGAME_DIM_LABELS,
  type KunGalgameDim
} from '~/constants/galgame-rating'

const props = withDefaults(
  defineProps<{
    modelValue: Record<KunGalgameDim, number>
    max?: number
    size?: number
    readonly?: boolean
    labelClass?: string
  }>(),
  {
    max: 10,
    size: 240,
    readonly: false,
    labelClass: ''
  }
)
const emit = defineEmits<{
  'update:modelValue': [Record<KunGalgameDim, number>]
}>()

const axes = KUN_GALGAME_DIMENSIONS.map((dim) => ({
  key: dim,
  label: KUN_GALGAME_DIM_LABELS[dim]
}))
type AxisKey = (typeof axes)[number]['key']

const center = computed(() => ({ x: props.size / 2, y: props.size / 2 }))
const radius = computed(() => props.size / 2 - 20)
const values = ref(props.modelValue)
watch(
  () => props.modelValue,
  () => (values.value = props.modelValue)
)

const angleOf = (i: number) => -Math.PI / 2 + (i * Math.PI * 2) / axes.length
const round1 = (n: number) => Math.round(n * 10) / 10
const polar = (r: number, i: number) => ({
  x: round1(center.value.x + r * Math.cos(angleOf(i))),
  y: round1(center.value.y + r * Math.sin(angleOf(i)))
})
const pointsOf = (radiusAt: (i: number) => number) =>
  axes
    .map((_, i) => {
      const { x, y } = polar(radiusAt(i), i)
      return `${x},${y}`
    })
    .join(' ')

const points = computed(() =>
  pointsOf((i) => {
    const raw = values.value[axes[i]!.key] ?? 0
    return (Math.max(0, Math.min(props.max, raw)) / props.max) * radius.value
  })
)

const ringPath = computed(() =>
  Array.from(
    { length: props.max },
    (_, level) =>
      `M${pointsOf(() => ((level + 1) / props.max) * radius.value)}Z`
  ).join('')
)

const spokes = computed(() =>
  axes.map((axis, i) => ({
    key: axis.key,
    label: axis.label,
    end: polar(radius.value, i),
    labelAt: polar(radius.value + 10, i)
  }))
)
const spokePath = computed(() =>
  spokes.value
    .map(({ end }) => `M${center.value.x},${center.value.y}L${end.x},${end.y}`)
    .join('')
)

const onClick = (e: MouseEvent) => {
  if (props.readonly) return
  const rect = (e.currentTarget as SVGElement).getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const dx = x - center.value.x
  const dy = y - center.value.y
  const angle = Math.atan2(dy, dx)
  const idx =
    Math.round(
      ((angle + Math.PI / 2 + Math.PI * 2) % (Math.PI * 2)) /
        ((Math.PI * 2) / axes.length)
    ) % axes.length
  const dist = Math.sqrt(dx * dx + dy * dy)
  const val = Math.max(
    0,
    Math.min(props.max, Math.round((dist / radius.value) * props.max))
  )
  const key = axes[idx] ? axes[idx].key : ('' as AxisKey)
  const next = { ...values.value, [key]: val }
  emit('update:modelValue', next)
}
</script>

<template>
  <div class="flex flex-col items-center space-y-2 select-none">
    <svg
      :width="size"
      :height="size"
      @click="onClick"
      :class="readonly ? 'cursor-default' : 'cursor-pointer'"
    >
      <path
        :d="ringPath"
        fill="none"
        class="stroke-default-200"
        stroke-width="0.5"
      />
      <path :d="spokePath" class="stroke-default-300" stroke-width="0.5" />

      <polygon
        :points="points"
        class="fill-secondary/30 stroke-secondary"
        stroke-width="1.5"
      />

      <text
        v-for="spoke in spokes"
        :key="spoke.key"
        :x="spoke.labelAt.x"
        :y="spoke.labelAt.y"
        text-anchor="middle"
        alignment-baseline="middle"
        :class="cn('fill-current text-xs', labelClass)"
      >
        {{ spoke.label }}
      </text>
    </svg>
  </div>
</template>
