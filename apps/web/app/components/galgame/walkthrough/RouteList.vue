<script setup lang="ts">
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
}>()

const selectedRoute = defineModel<string | null>('selectedRoute', {
  required: true
})

const nameOf = useCatalogName()

const routes = computed(() => buildRoutes(props.chart, props.characters))
const tones = computed(() => characterTones(props.characters))
const creators = computed(() => saveCreators(props.chart))

const current = computed(
  () =>
    routes.value.find((r) => r.key === selectedRoute.value) ?? routes.value[0]
)

const characterOf = (key: string) => props.characters.find((c) => c.key === key)

const displayName = (key: string, fallback: string) => {
  const ref = characterOf(key)?.catalog_character
  return ref ? nameOf(ref.catalog_name).name : fallback
}

const toneClass = (tone: number | null) =>
  tone === null
    ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE
    : KUN_GALGAME_WALKTHROUGH_TONES[
        tone % KUN_GALGAME_WALKTHROUGH_TONES.length
      ]!

const endingCount = (key: string) =>
  routes.value
    .find((r) => r.key === key)
    ?.segments.filter((s) => s.to_node.node_type === 'ending').length ?? 0

const highlightedSave = ref<string | null>(null)
let clearHighlight: ReturnType<typeof setTimeout> | undefined

const jump = async (saveKey: string) => {
  const segmentKey = creators.value.get(saveKey)
  const owner = routes.value.find((r) =>
    r.segments.some((s) => s.key === segmentKey)
  )
  if (owner) selectedRoute.value = owner.key
  await nextTick()
  document
    .getElementById(`wt-save-${saveKey}`)
    ?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  highlightedSave.value = saveKey
  clearTimeout(clearHighlight)
  clearHighlight = setTimeout(() => (highlightedSave.value = null), 1800)
}
</script>

<template>
  <div class="grid gap-4 lg:grid-cols-[15rem_minmax(0,1fr)]">
    <nav
      class="flex gap-2 overflow-x-auto pb-1 lg:sticky lg:top-24 lg:flex-col lg:self-start lg:overflow-visible"
      aria-label="攻略角色"
    >
      <KunButton
        v-for="route in routes"
        :key="route.key"
        :variant="current?.key === route.key ? 'flat' : 'light'"
        :color="current?.key === route.key ? 'primary' : 'default'"
        class-name="shrink-0 justify-start"
        @click="selectedRoute = route.key"
      >
        <span
          :class="
            cn('size-2.5 shrink-0 rounded-full', toneClass(route.tone).dot)
          "
        />
        <span class="truncate">{{ displayName(route.key, route.name) }}</span>
        <span class="text-default-400 ml-auto pl-2 text-xs">
          {{ endingCount(route.key) }} 结局
        </span>
      </KunButton>
    </nav>

    <div v-if="current" class="min-w-0 space-y-3">
      <GalgameWalkthroughSegmentCard
        v-for="segment in current.segments"
        :key="segment.key"
        :segment="segment"
        :chart="chart"
        :tones="tones"
        :highlighted-save="highlightedSave"
        @jump="jump"
      />
    </div>
    <KunNull v-else description="这份攻略没有可显示的路线" />
  </div>
</template>
