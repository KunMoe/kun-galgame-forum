<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import type { CompanyGraph } from '#shared/utils/api/schemas'

const props = defineProps<{
  officialId: number
}>()

const nameOf = useCatalogName()

const { data: v1Graph } = useApi<CompanyGraph>(
  () => `company-graph:${props.officialId}`,
  (api) =>
    api.GET('/companies/{company_id}/graph', {
      params: { path: { company_id: String(props.officialId) } }
    }),
  { lazy: true }
)

const data = computed<GalgameOfficialRelationGraph | undefined>(() =>
  v1Graph.value
    ? {
        nodes: v1Graph.value.nodes.map((n) => ({
          id: Number(n.id),
          name: nameOf(n).name,
          logo: n.logo?.url ?? '',
          work_count: n.catalog_work_count
        })),
        edges: v1Graph.value.edges.map((e) => ({
          from: Number(e.from_company_id),
          to: Number(e.to_company_id),
          relation: e.relation
        }))
      }
    : undefined
)

const graph = computed(() =>
  (data.value?.nodes.length ?? 0) > 1 ? data.value : null
)

const layout = computed(() =>
  graph.value ? buildOfficialGraphLayout(graph.value) : null
)

const hasContent = computed(() => (layout.value?.edges.length ?? 0) > 0)

const view = ref('graph')
const tabs: KunTabItem[] = [
  { value: 'graph', textValue: '关系图', icon: 'lucide:git-fork' },
  { value: 'list', textValue: '列表', icon: 'lucide:list-tree' }
]

const selectedId = ref<number | null>(props.officialId)

const isFullscreen = ref(false)

const open = (id: number) => {
  isFullscreen.value = false
  return navigateTo(taxonomyDetailPath('official', id))
}
</script>

<template>
  <div v-if="hasContent && graph && layout" class="space-y-4">
    <KunHeader
      name="会社关系"
      description="该会社所属的企业家族: 母公司、旗下品牌、更名沿革与拆分出的公司。资料来自 NextMoe 目录。"
      scale="h3"
    >
      <template #headerEndContent>
        <KunTab v-model="view" :items="tabs" variant="light" size="sm" />
      </template>
    </KunHeader>

    <KunTabPanels v-model="view">
      <KunTabPanel
        value="graph"
        class-name="grid gap-3 lg:grid-cols-[minmax(0,1fr)_17rem]"
      >
        <ClientOnly>
          <GalgameOfficialGraphCanvas
            v-model:selected-id="selectedId"
            :layout="layout"
            :current-id="officialId"
            @open="open"
            @expand="isFullscreen = true"
          />
          <template #fallback>
            <div
              class="border-default-200 bg-default-50 h-48 rounded-xl border sm:h-[32rem]"
            />
          </template>
        </ClientOnly>

        <GalgameOfficialGraphInspector
          :layout="layout"
          :current-id="officialId"
          :selected-id="selectedId"
          @select="selectedId = $event"
        />
      </KunTabPanel>

      <KunTabPanel value="list">
        <GalgameOfficialRelationLanes
          :graph="graph"
          :official-id="officialId"
        />
      </KunTabPanel>
    </KunTabPanels>

    <KunModal
      v-model="isFullscreen"
      inner-class-name="w-[96vw] h-[92vh] max-w-none"
      aria-label="会社关系图"
    >
      <div
        v-if="isFullscreen"
        class="flex h-full flex-col gap-3 lg:flex-row lg:gap-4"
      >
        <div class="min-h-0 flex-1">
          <GalgameOfficialGraphCanvas
            v-model:selected-id="selectedId"
            :layout="layout"
            :current-id="officialId"
            :is-fullscreen="true"
            @open="open"
          />
        </div>
        <div
          class="max-h-[30vh] shrink-0 overflow-y-auto lg:max-h-none lg:w-72"
        >
          <GalgameOfficialGraphInspector
            :layout="layout"
            :current-id="officialId"
            :selected-id="selectedId"
            @select="selectedId = $event"
          />
        </div>
      </div>
    </KunModal>
  </div>
</template>
