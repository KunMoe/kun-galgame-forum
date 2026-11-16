<script setup lang="ts">
const props = defineProps<{
  officialId: number
  graph: GalgameOfficialRelationGraph
  layout: GalgameOfficialGraphLayout
  relatedCount: number
}>()

const isGraphOpen = ref(false)
const selectedId = ref<number | null>(props.officialId)

const openGraph = () => {
  selectedId.value = props.officialId
  isGraphOpen.value = true
}

const open = (id: number) => {
  isGraphOpen.value = false
  return navigateTo(taxonomyDetailPath('official', id))
}
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="false"
    padding="md"
    content-class="gap-4 justify-start"
  >
    <div class="space-y-1">
      <div class="flex items-center justify-between gap-2">
        <h2 class="text-lg font-semibold">会社关系</h2>
        <span class="text-default-400 text-xs">关联 {{ relatedCount }} 家</span>
      </div>
      <p class="text-default-500 text-xs">
        母公司、旗下品牌、更名沿革与拆分出的公司
      </p>
    </div>

    <KunButton
      variant="flat"
      color="primary"
      size="sm"
      :full-width="true"
      @click="openGraph"
    >
      <KunIcon name="lucide:git-fork" />
      查看关系图
    </KunButton>

    <GalgameOfficialRelationLanes :graph="graph" :official-id="officialId" />

    <KunModal
      v-model="isGraphOpen"
      inner-class-name="w-[96vw] h-[92vh] max-w-none"
      aria-label="会社关系图"
    >
      <div
        v-if="isGraphOpen"
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
  </KunCard>
</template>
