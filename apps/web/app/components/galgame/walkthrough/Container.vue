<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import type { Walkthrough, Work } from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_WALKTHROUGH_NEUTRAL_TONE,
  KUN_GALGAME_WALKTHROUGH_SOURCE,
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
const workName = useWorkName()

const chart = computed(() => props.walkthrough.chart!)
const characters = computed(() => props.walkthrough.chart_characters)

const textMode = ref<KunWalkthroughTextMode>('both')
provide('walkthroughTextMode', textMode)

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
const tones = computed(() => characterTones(characters.value))
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

const source = computed(
  () =>
    KUN_GALGAME_WALKTHROUGH_SOURCE[props.walkthrough.source_site] ?? {
      name: props.walkthrough.source_site,
      home: props.walkthrough.source_page_url
    }
)

const characterName = (c: (typeof characters.value)[number]) =>
  c.catalog_character
    ? nameOf(c.catalog_character.catalog_name).name
    : c.source_name
</script>

<template>
  <div class="space-y-4">
    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <KunHeader
        :name="`${workName(work)} 攻略`"
        :description="`${walkthrough.ending_count} 个结局 · ${walkthrough.save_count} 个存档点 · ${characters.length} 名角色`"
      >
        <template #headerEndContent>
          <KunButton :href="`/galgame/${work.id}`" variant="light" size="sm">
            <KunIcon name="lucide:arrow-left" />
            返回游戏页面
          </KunButton>
        </template>
      </KunHeader>

      <div class="flex flex-wrap gap-2">
        <KunButton
          v-for="c in characters"
          :key="c.key"
          size="sm"
          :variant="selectedRoute === c.key ? 'flat' : 'bordered'"
          :color="selectedRoute === c.key ? 'primary' : 'default'"
          @click="selectedRoute = c.key"
        >
          <span
            :class="
              cn('size-2 rounded-full', toneClass(tones.get(c.key) ?? null).dot)
            "
          />
          {{ characterName(c) }}
        </KunButton>
      </div>

      <KunInfo color="default" title="关于这份攻略">
        <p class="text-default-600 text-sm">
          选项流程整理自
          <KunLink size="sm" :to="walkthrough.source_page_url" target="_blank">
            {{ source.name }}
          </KunLink>
          <template v-if="walkthrough.source_author">
            (作者 {{ walkthrough.source_author }})</template
          >
          <template v-if="walkthrough.source_updated_date"
            >, 原页面更新于 {{ walkthrough.source_updated_date }}</template
          >
          。只保留了选项与存档结构, 原文说明请查看原页面。选项的中文为机器翻译,
          游戏内显示的是日文原文; 没有可靠译名的人名保持原文。
        </p>
      </KunInfo>
    </KunCard>

    <div class="flex flex-wrap items-center justify-between gap-3">
      <KunTab v-model="view" :items="views" variant="solid" size="sm" />
      <div class="flex items-center gap-3">
        <KunSwitch
          v-if="view === 'graph'"
          v-model="highlightRoute"
          label="只看所选角色"
        />
        <KunTab
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
        />
      </KunTabPanel>
      <KunTabPanel
        value="graph"
        mount="lazy"
        class-name="grid gap-3 lg:grid-cols-[minmax(0,1fr)_18rem]"
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
              class="border-default-200 bg-default-50 h-[28rem] rounded-xl border sm:h-[36rem]"
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
