<script setup lang="ts">
const props = defineProps<{ cards: AgentEntityCard[] }>()

const { coverOf } = useAgentCoverGate()
const preview = useAgentPreview()

const tags = computed(() => agentTagChips(props.cards))
const items = computed(() =>
  props.cards
    .filter((c) => c.kind !== 'tag')
    .map((c) => ({
      key: agentCardKey(c),
      card: c,
      previewable: agentCardPreviewable(c),
      title: agentCardTitle(c),
      caption: c.subtitle || c.release_date?.slice(0, 4) || '',
      cover: coverOf(c)
    }))
)
</script>

<template>
  <div class="flex flex-col gap-2">
    <div v-if="items.length" class="-mx-1 flex gap-2 overflow-x-auto px-1 pb-1">
      <component
        :is="item.previewable ? 'button' : 'div'"
        v-for="item in items"
        :key="item.key"
        :type="item.previewable ? 'button' : undefined"
        :aria-label="item.previewable ? `预览「${item.title}」` : undefined"
        :class="
          cn(
            'group border-default-200 bg-content1 rounded-kun-md w-28 shrink-0 overflow-hidden border text-left',
            item.previewable && 'cursor-pointer'
          )
        "
        @click="item.previewable && preview.open(item.card)"
      >
        <div class="bg-default-100 relative aspect-[3/4] overflow-hidden">
          <AgentCover :title="item.title" :cover="item.cover" />
        </div>
        <div class="p-1.5">
          <p
            :class="
              cn(
                'line-clamp-2 text-xs leading-snug font-medium',
                item.previewable && 'group-hover:text-primary'
              )
            "
          >
            {{ item.title }}
          </p>
          <p
            v-if="item.caption"
            class="text-default-400 mt-0.5 line-clamp-1 text-[11px]"
          >
            {{ item.caption }}
          </p>
        </div>
      </component>
    </div>
    <AgentTagChips v-if="tags.length" :tags="tags" />
  </div>
</template>
