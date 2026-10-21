<script setup lang="ts">
import { KUN_GALGAME_OFFICIAL_RELATION_MAP } from '~/constants/galgameOfficial'

const props = defineProps<{
  layout: GalgameOfficialGraphLayout
  currentId: number
  selectedId: number | null
}>()

const emit = defineEmits<{
  select: [id: number]
}>()

const nodeOf = (id: number) =>
  props.layout.nodes.find((n) => n.official.id === id)?.official

const selected = computed(() =>
  props.selectedId === null ? undefined : nodeOf(props.selectedId)
)

const OUTGOING: Record<string, string> = {
  subsidiary: 'subsidiary',
  imprint: 'imprint',
  succession: 'succeeded_by',
  spawn: 'spawned'
}
const INCOMING: Record<string, string> = {
  subsidiary: 'parent',
  imprint: 'imprint_of',
  succession: 'formerly',
  spawn: 'origin'
}

interface RelationGroup {
  label: string
  officials: GalgameOfficialRelationNode[]
}

const groups = computed<RelationGroup[]>(() => {
  const id = props.selectedId
  if (id === null) return []

  const byLabel = new Map<string, GalgameOfficialRelationNode[]>()
  const add = (key: string, other: number) => {
    const official = nodeOf(other)
    if (!official) return
    const label = KUN_GALGAME_OFFICIAL_RELATION_MAP[key] ?? key
    byLabel.set(label, [...(byLabel.get(label) ?? []), official])
  }

  for (const edge of props.layout.edges) {
    if (edge.from === id) add(OUTGOING[edge.kind]!, edge.to)
    else if (edge.to === id) add(INCOMING[edge.kind]!, edge.from)
  }

  return [...byLabel].map(([label, officials]) => ({ label, officials }))
})

const logoSrc = computed(() =>
  selected.value?.logo ? withImageVariant(selected.value.logo, 'mini') : ''
)

const LEGEND = [
  { klass: 'bg-default-300', text: '所属 / 旗下', dashed: false },
  { klass: 'bg-secondary-300', text: '更名', dashed: true },
  { klass: 'bg-warning-300', text: '拆分', dashed: true }
]
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="false"
    padding="md"
    content-class="gap-3 justify-start"
  >
    <template v-if="selected">
      <div class="flex items-center gap-2">
        <div v-if="logoSrc" class="bg-default-100 shrink-0 rounded-md p-1">
          <KunImage
            :src="logoSrc"
            :alt="`${selected.name} logo`"
            object-fit="contain"
            class-name="size-8"
          />
        </div>
        <div class="min-w-0">
          <p class="truncate font-medium">{{ selected.name }}</p>
          <p v-if="selected.work_count > 0" class="text-default-400 text-xs">
            {{ `收录 ${selected.work_count} 部` }}
          </p>
        </div>
      </div>

      <div v-if="groups.length" class="space-y-2">
        <div v-for="group in groups" :key="group.label" class="space-y-1">
          <p class="text-default-500 text-xs">{{ group.label }}</p>
          <div class="flex flex-wrap gap-1">
            <KunButton
              v-for="official in group.officials"
              :key="official.id"
              variant="flat"
              :color="official.id === currentId ? 'primary' : 'default'"
              size="sm"
              @click="emit('select', official.id)"
            >
              {{ official.name }}
            </KunButton>
          </div>
        </div>
      </div>

      <KunButton
        v-if="selected.id !== currentId"
        variant="flat"
        color="primary"
        size="sm"
        :full-width="true"
        :href="taxonomyDetailPath('official', selected.id)"
      >
        <KunIcon name="lucide:arrow-right" />
        前往 {{ selected.name }}
      </KunButton>
      <p v-else class="text-default-400 text-xs">你正在浏览这家会社。</p>
    </template>

    <template v-else>
      <p class="text-default-500 text-sm">
        点击图中的任意会社查看它的关系, 双击或再次点击前往它的页面。
      </p>
      <div class="space-y-1.5">
        <div
          v-for="item in LEGEND"
          :key="item.text"
          class="text-default-500 flex items-center gap-2 text-xs"
        >
          <span
            :class="
              cn(
                'h-0.5 w-6 shrink-0 rounded-full',
                item.klass,
                item.dashed && 'opacity-70'
              )
            "
          />
          {{ item.text }}
        </div>
      </div>
    </template>
  </KunCard>
</template>
