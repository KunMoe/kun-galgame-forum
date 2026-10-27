<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import type { Walkthrough, Work } from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE,
  KUN_GALGAME_WALKTHROUGH_TEXT_MODES,
  KUN_GALGAME_WALKTHROUGH_TONES,
  type KunWalkthroughTextMode
} from '~/constants/galgameWalkthrough'

const props = defineProps<{
  walkthrough: Walkthrough
  work: Work
}>()

const route = useRoute()
const writeQuery = useQueryWriter()
const nameOf = useCatalogName()

const chart = computed(() => props.walkthrough.chart!)
const characters = computed(() => props.walkthrough.chart_characters)

const textMode = ref<KunWalkthroughTextMode>('both')
provide('walkthroughTextMode', textMode)
const textLang = computed(() => props.walkthrough.text_lang)
provide('walkthroughTextLang', textLang)

const view = ref(route.query.view === 'graph' ? 'graph' : 'list')
const views: KunTabItem[] = [
  { value: 'list', textValue: '路线列表', icon: 'lucide:list-tree' },
  { value: 'graph', textValue: '流程图', icon: 'lucide:git-fork' }
]
const textModes: KunTabItem[] = KUN_GALGAME_WALKTHROUGH_TEXT_MODES.map((m) => ({
  value: m.value,
  textValue: m.textValue
}))

const routes = computed(() => buildRoutes(chart.value, characters.value))
const portraits = computed(() =>
  walkthroughPortraits(characters.value, props.work.roster)
)
const initialRoute = () => {
  const q = route.query.character
  return typeof q === 'string' && routes.value.some((r) => r.key === q)
    ? q
    : (routes.value[0]?.key ?? null)
}
const selectedRoute = ref<string | null>(initialRoute())
const highlightRoute = ref(false)

watch([view, selectedRoute], ([v, r]) => {
  writeQuery({
    view: v === 'graph' ? 'graph' : undefined,
    character: r ?? undefined
  })
})

const layout = computed(() =>
  layoutWalkthroughGraph(buildGraphModel(chart.value, characters.value))
)
const routeSegments = computed(() => {
  if (!highlightRoute.value) return null
  const r = routes.value.find((x) => x.key === selectedRoute.value)
  return r ? new Set(r.segments.map((s) => s.key)) : null
})
const selected = ref<{ kind: 'node' | 'edge'; id: string } | null>(null)
const isFullscreen = ref(false)

const openSegment = async (segmentKey: string) => {
  const owner = routes.value.find((r) =>
    r.segments.some((s) => s.key === segmentKey)
  )
  if (owner) selectedRoute.value = owner.key
  isFullscreen.value = false
  view.value = 'list'
  await nextTick()
  document
    .getElementById(`wt-seg-${segmentKey}`)
    ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const toneClass = (tone: number | null) =>
  tone === null
    ? KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE
    : KUN_GALGAME_WALKTHROUGH_TONES[
        tone % KUN_GALGAME_WALKTHROUGH_TONES.length
      ]!

const routeName = (key: string, fallback: string) => {
  const ref = characters.value.find((c) => c.key === key)?.catalog_character
  return ref ? nameOf(ref.catalog_name).name : fallback
}

const endingCount = (segments: (typeof routes.value)[number]['segments']) =>
  segments.filter((s) => s.to_node.node_type === 'ending').length
</script>

<template>
  <div class="space-y-6">
    <section class="space-y-3" aria-label="角色路线">
      <div class="flex items-baseline justify-between gap-3">
        <h2 class="text-foreground text-lg font-semibold">角色路线</h2>
        <span class="text-default-500 text-xs">选择角色查看这条路线的选项</span>
      </div>
      <div
        class="scrollbar-hide -mx-1 flex gap-3 overflow-x-auto px-1 pt-1 pb-2 sm:flex-wrap sm:overflow-visible"
      >
        <button
          v-for="r in routes"
          :key="r.key"
          type="button"
          :aria-pressed="selectedRoute === r.key"
          class="group w-24 shrink-0 space-y-1.5 text-left focus:outline-none sm:w-28"
          @click="selectedRoute = r.key"
        >
          <div
            :class="
              cn(
                'rounded-xl ring-2 ring-offset-2 ring-offset-transparent transition-shadow',
                selectedRoute === r.key
                  ? 'ring-primary-500'
                  : 'group-hover:ring-default-300 group-focus-visible:ring-default-300 ring-transparent'
              )
            "
          >
            <GalgameWalkthroughPortrait
              :image="portraits.get(r.key)"
              :name="routeName(r.key, r.name)"
              :icon="
                r.key === COMMON_ROUTE_KEY ? 'lucide:git-merge' : undefined
              "
              size="bust"
            />
          </div>
          <div class="flex items-center gap-1.5">
            <span
              :class="cn('size-2 shrink-0 rounded-full', toneClass(r.tone).dot)"
            />
            <span
              :class="
                cn(
                  'truncate text-sm',
                  selectedRoute === r.key
                    ? 'text-primary-600 font-semibold'
                    : 'text-foreground'
                )
              "
            >
              {{ routeName(r.key, r.name) }}
            </span>
          </div>
          <p class="text-default-500 text-xs">
            {{ endingCount(r.segments) }} 个结局 · {{ r.segments.length }} 段
          </p>
        </button>
      </div>
    </section>

    <div class="flex flex-wrap items-center justify-between gap-3">
      <KunTab v-model="view" :items="views" variant="solid" size="sm" />
      <div class="flex flex-wrap items-center gap-3">
        <KunSwitch
          v-if="view === 'graph'"
          v-model="highlightRoute"
          label="只看所选角色"
        />
        <KunTab
          v-if="textLang !== 'zh-Hans'"
          v-model="textMode"
          :items="textModes"
          variant="light"
          size="sm"
        />
      </div>
    </div>

    <KunTabPanels v-model="view">
      <KunTabPanel value="list">
        <GalgameWalkthroughRouteList
          v-model:selected-route="selectedRoute"
          :chart="chart"
          :characters="characters"
          :portraits="portraits"
        />
      </KunTabPanel>
      <KunTabPanel
        value="graph"
        mount="lazy"
        class-name="grid gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]"
      >
        <ClientOnly>
          <GalgameWalkthroughGraph
            v-model:selected="selected"
            :layout="layout"
            :route-segments="routeSegments"
            @expand="isFullscreen = true"
          />
          <template #fallback>
            <div
              class="border-default-200 bg-default-50 h-[36rem] rounded-xl border sm:h-[44rem]"
            />
          </template>
        </ClientOnly>
        <GalgameWalkthroughInspector
          :layout="layout"
          :chart="chart"
          :selected="selected"
          @open-segment="openSegment"
        />
      </KunTabPanel>
    </KunTabPanels>

    <KunModal
      v-model="isFullscreen"
      inner-class-name="w-[96vw] h-[92vh] max-w-none"
      aria-label="攻略流程图"
    >
      <div v-if="isFullscreen" class="flex h-full flex-col gap-3 lg:flex-row">
        <div class="min-h-0 flex-1">
          <GalgameWalkthroughGraph
            v-model:selected="selected"
            :layout="layout"
            :route-segments="routeSegments"
            :is-fullscreen="true"
          />
        </div>
        <div
          class="max-h-[30vh] shrink-0 overflow-y-auto lg:max-h-none lg:w-72"
        >
          <GalgameWalkthroughInspector
            :layout="layout"
            :chart="chart"
            :selected="selected"
            @open-segment="openSegment"
          />
        </div>
      </div>
    </KunModal>
  </div>
</template>
