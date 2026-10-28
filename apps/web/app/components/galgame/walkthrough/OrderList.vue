<script setup lang="ts">
import { useLocalStorage } from '@vueuse/core'
import type {
  WalkthroughCharacter,
  WalkthroughChart
} from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE,
  KUN_GALGAME_WALKTHROUGH_TONES
} from '~/constants/galgameWalkthrough'

const props = defineProps<{
  chart: WalkthroughChart
  characters: WalkthroughCharacter[]
  walkthroughId: string
}>()

const labels = useWalkthroughLabels()
const routeName = inject<(key: string) => string>(
  'walkthroughRouteName',
  (key) => key
)

const tones = computed(() => characterTones(props.characters))
const creators = computed(() => saveCreators(props.chart))

const reachedAt = computed(() => {
  const out = new Map<string, number>()
  props.chart.chart_segments.forEach((s, i) => {
    if (s.to_node.node_type === 'ending' && s.to_node.node_key) {
      out.set(s.to_node.node_key, i)
    }
  })
  return out
})

const items = computed(() =>
  props.chart.chart_segments.map((segment) => {
    const key = segmentCharacterKey(props.chart, segment)
    const tone = key ? (tones.value.get(key) ?? null) : null
    return {
      segment,
      requires: requiredEndingKeys(segment).flatMap((k) => {
        const at = reachedAt.value.get(k)
        return at === undefined ? [] : [{ key: k, at }]
      }),
      route: key ? routeName(key) : '共通',
      dot:
        tone === null
          ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE.dot
          : KUN_GALGAME_WALKTHROUGH_TONES[
              tone % KUN_GALGAME_WALKTHROUGH_TONES.length
            ]!.dot
    }
  })
)

const done = useLocalStorage<string[]>(
  `kun-walkthrough-done:${props.walkthroughId}`,
  [],
  { initOnMounted: true }
)
const doneSet = computed(() => new Set(done.value))

const open = ref<string[]>([])

const setDone = (segmentKey: string, value: boolean) => {
  if (!value) {
    done.value = done.value.filter((k) => k !== segmentKey)
    return
  }
  done.value = [...new Set([...done.value, segmentKey])]
  const next = items.value.find((i) => !doneSet.value.has(i.segment.key))
  open.value = [
    ...open.value.filter((k) => k !== segmentKey),
    ...(next ? [next.segment.key] : [])
  ]
}

const highlightedSave = ref<string | null>(null)
let clearHighlight: ReturnType<typeof setTimeout> | undefined

const jump = async (saveKey: string) => {
  const segmentKey = creators.value.get(saveKey)
  const target = document.getElementById(`wt-order-save-${saveKey}`)
  if (segmentKey && !open.value.includes(segmentKey)) {
    open.value = [...open.value, segmentKey]
    // The accordion grows its panel with a CSS transition; scrolling earlier
    // lands where the save sat while the panel was still collapsed.
    const panel = target?.closest('[role="region"]')
    const ms = panel
      ? parseFloat(getComputedStyle(panel).transitionDuration) * 1000
      : 0
    await new Promise((resolve) => setTimeout(resolve, ms))
  }
  target?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  highlightedSave.value = saveKey
  clearTimeout(clearHighlight)
  clearHighlight = setTimeout(() => (highlightedSave.value = null), 1800)
}
</script>

<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
        <p class="text-default-500 text-sm">
          按原攻略的顺序依次游玩, 完成一段后勾选, 进度只保存在这台设备的浏览器里
        </p>
        <span class="text-foreground ml-auto text-sm">
          已完成 {{ doneSet.size }} / {{ items.length }} 段
        </span>
        <KunButton
          v-if="doneSet.size"
          size="sm"
          variant="light"
          color="default"
          @click="done = []"
        >
          清除进度
        </KunButton>
      </div>
      <KunProgress
        :value="doneSet.size"
        :max="items.length"
        size="sm"
        color="success"
        aria-label="攻略进度"
      />
    </div>

    <KunAccordion v-model="open" :multiple="true" variant="splitted">
      <KunAccordionItem
        v-for="(item, i) in items"
        :key="item.segment.key"
        :value="item.segment.key"
      >
        <template #title>
          <div class="flex min-w-0 items-center gap-3 text-left">
            <span
              :class="
                cn(
                  'flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-medium',
                  doneSet.has(item.segment.key)
                    ? 'bg-success-500 text-white'
                    : 'bg-default-100 text-default-600'
                )
              "
            >
              <KunIcon
                v-if="doneSet.has(item.segment.key)"
                name="lucide:check"
                class-name="size-4"
              />
              <template v-else>{{ i + 1 }}</template>
            </span>
            <div class="min-w-0 space-y-0.5">
              <div class="flex flex-wrap items-center gap-x-2 text-sm">
                <span class="text-default-600">
                  {{ labels.start(chart, item.segment) }}
                </span>
                <KunIcon
                  name="lucide:arrow-right"
                  class-name="text-default-400 size-3.5"
                />
                <span class="text-foreground font-medium">
                  {{ labels.goal(chart, item.segment) }}
                </span>
              </div>
              <div
                class="text-default-500 flex flex-wrap items-center gap-x-3 text-xs"
              >
                <span class="flex items-center gap-1.5">
                  <span :class="cn('size-2 rounded-full', item.dot)" />
                  {{ item.route }}
                </span>
                <span>{{ choiceCount(item.segment.steps) }} 个选项</span>
                <span
                  v-for="r in item.requires"
                  :key="r.key"
                  :class="
                    doneSet.has(chart.chart_segments[r.at]!.key)
                      ? 'text-success-600'
                      : 'text-warning-600'
                  "
                >
                  需先看过 {{ labels.ending(chart, r.key) }} (第
                  {{ r.at + 1 }} 段)
                </span>
                <span
                  v-if="item.segment.condition && !item.requires.length"
                  class="text-warning-600"
                >
                  有前置条件
                </span>
              </div>
            </div>
          </div>
        </template>

        <div class="space-y-4">
          <GalgameWalkthroughSegment
            :segment="item.segment"
            :chart="chart"
            :tones="tones"
            :highlighted-save="highlightedSave"
            id-prefix="wt-order"
            @jump="jump"
          />
          <KunCheckBox
            :model-value="doneSet.has(item.segment.key)"
            label="这一段已经打完"
            @update:model-value="setDone(item.segment.key, $event)"
          />
        </div>
      </KunAccordionItem>
    </KunAccordion>
  </div>
</template>
