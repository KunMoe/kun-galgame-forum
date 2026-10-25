<script setup lang="ts">
import type {
  Image,
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
  portraits: Map<string, Image>
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
  <div class="grid gap-6 lg:grid-cols-[13rem_minmax(0,1fr)]">
    <nav
      class="hidden flex-col gap-1 lg:sticky lg:top-24 lg:flex lg:self-start"
      aria-label="攻略角色"
    >
      <KunButton
        v-for="route in routes"
        :key="route.key"
        :variant="current?.key === route.key ? 'flat' : 'light'"
        :color="current?.key === route.key ? 'primary' : 'default'"
        class-name="justify-start"
        @click="selectedRoute = route.key"
      >
        <GalgameWalkthroughPortrait
          :image="portraits.get(route.key)"
          :name="displayName(route.key, route.name)"
          :icon="
            route.key === COMMON_ROUTE_KEY ? 'lucide:git-merge' : undefined
          "
          size="avatar"
        />
        <span class="truncate">{{ displayName(route.key, route.name) }}</span>
        <span
          :class="
            cn(
              'ml-auto size-2 shrink-0 rounded-full',
              toneClass(route.tone).dot
            )
          "
        />
      </KunButton>
    </nav>

    <div v-if="current" class="min-w-0 space-y-8">
      <GalgameWalkthroughSegment
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
