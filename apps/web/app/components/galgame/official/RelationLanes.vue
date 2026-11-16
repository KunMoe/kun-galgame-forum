<script setup lang="ts">
const props = defineProps<{
  graph: GalgameOfficialRelationGraph
  officialId: number
}>()

const forest = computed(() =>
  buildOfficialFamilyForest(props.graph, props.officialId)
)
const renameChains = computed(() => buildOfficialRenameChains(props.graph))

const spawnRows = computed(() =>
  buildOfficialSpawnPairs(props.graph).map((pair) => {
    if (pair.parent.id === props.officialId) {
      return { prefix: '拆分出', official: pair.child, suffix: '' }
    }
    if (pair.child.id === props.officialId) {
      return { prefix: '从', official: pair.parent, suffix: '拆分而来' }
    }
    return {
      prefix: `${pair.parent.name} 拆分出`,
      official: pair.child,
      suffix: ''
    }
  })
)
</script>

<template>
  <div class="space-y-4">
    <div v-if="forest.length" class="space-y-3">
      <h4 class="text-default-500 text-sm">企业家族</h4>
      <GalgameOfficialRelationTree
        v-for="root in forest"
        :key="root.official.id"
        :nodes="[root]"
        :current-id="officialId"
      />
    </div>

    <div v-if="renameChains.length" class="space-y-3">
      <h4 class="text-default-500 text-sm">更名沿革</h4>
      <div
        v-for="(chain, index) in renameChains"
        :key="index"
        class="flex flex-wrap items-center gap-2"
      >
        <template v-for="(item, step) in chain" :key="item.id">
          <KunIcon
            v-if="step"
            name="lucide:arrow-right"
            class-name="text-default-400 shrink-0"
          />
          <span
            v-if="item.id === officialId"
            class="border-primary-500 bg-primary-50 text-primary-600 shrink-0 rounded-md border px-2 py-1 text-sm font-medium"
          >
            {{ item.name }}
          </span>
          <KunButton
            v-else
            variant="flat"
            color="default"
            size="sm"
            class-name="shrink-0"
            :href="taxonomyDetailPath('official', item.id)"
          >
            {{ item.name }}
          </KunButton>
        </template>
      </div>
    </div>

    <div v-if="spawnRows.length" class="space-y-3">
      <h4 class="text-default-500 text-sm">衍生</h4>
      <div
        v-for="(row, index) in spawnRows"
        :key="index"
        class="text-default-600 flex flex-wrap items-center gap-2 text-sm"
      >
        <KunIcon name="lucide:git-branch" class-name="text-default-400" />
        <span>{{ row.prefix }}</span>
        <span
          v-if="row.official.id === officialId"
          class="text-primary-600 font-medium"
        >
          {{ row.official.name }}
        </span>
        <KunLink
          v-else
          size="sm"
          underline="hover"
          :to="taxonomyDetailPath('official', row.official.id)"
        >
          {{ row.official.name }}
        </KunLink>
        <span v-if="row.suffix">{{ row.suffix }}</span>
      </div>
    </div>
  </div>
</template>
