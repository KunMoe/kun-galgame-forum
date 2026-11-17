<script setup lang="ts">
const props = defineProps<{ card: AgentEntityCard }>()

const MAX_TAGS = 3

const { coverOf } = useAgentCoverGate()
const preview = useAgentPreview()

const title = computed(() => agentCardTitle(props.card))
const cover = computed(() => coverOf(props.card))
const byline = computed(() =>
  [
    (props.card.companies ?? []).slice(0, 2).join('、'),
    props.card.release_date?.slice(0, 4)
  ]
    .filter(Boolean)
    .join(' · ')
)
const tags = computed(() => (props.card.tags ?? []).slice(0, MAX_TAGS))
const rating = computed(() => agentCardRating(props.card))
</script>

<template>
  <button
    type="button"
    :aria-label="`预览「${title}」`"
    class="group rounded-kun-md border-default-200 bg-content1 hover:bg-default-50 flex min-w-0 cursor-pointer gap-3 border p-2 text-left"
    @click="preview.open(card)"
  >
    <div
      class="bg-default-100 rounded-kun-md relative aspect-[3/4] w-24 shrink-0 overflow-hidden"
    >
      <AgentCover :title="title" :cover="cover" />
    </div>
    <div class="flex min-w-0 flex-1 flex-col gap-1.5">
      <p
        class="group-hover:text-primary line-clamp-2 text-sm leading-snug font-medium"
      >
        {{ title }}
      </p>
      <p v-if="byline" class="text-default-500 line-clamp-1 text-xs">
        {{ byline }}
      </p>
      <div v-if="tags.length" class="flex flex-wrap gap-1">
        <KunChip v-for="t in tags" :key="t" size="sm" variant="flat">
          {{ t }}
        </KunChip>
      </div>
      <p
        v-if="rating"
        class="text-warning-600 mt-auto flex items-center gap-1 text-xs font-medium"
      >
        <KunIcon name="lucide:star" class="icon-solid size-3.5" />
        {{ rating.score }}
        <span class="text-default-400 font-normal">{{ rating.label }}</span>
      </p>
    </div>
  </button>
</template>
