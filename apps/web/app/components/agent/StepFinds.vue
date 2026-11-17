<script setup lang="ts">
import { formatDate } from '#shared/utils/time'
const props = defineProps<{ output: AgentToolOutput }>()

const { coverOf } = useAgentCoverGate()
const preview = useAgentPreview()

const works = computed(() =>
  props.output.entities.filter((c) => c.kind === 'work')
)
const tags = computed(() => agentTagChips(props.output.entities))
const entities = computed(() =>
  props.output.entities
    .filter((c) => c.kind !== 'work' && c.kind !== 'tag')
    .map((c) => ({
      key: agentCardKey(c),
      card: c,
      previewable: agentCardPreviewable(c),
      title: agentCardTitle(c),
      caption: c.subtitle || c.release_date?.slice(0, 4) || '',
      cover: coverOf(c)
    }))
)

const past = computed(() => agentPastConversations(props.output.meta))

const hostOf = (url: string) => {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}
</script>

<template>
  <div class="flex flex-col gap-2 pt-1 pl-5">
    <div v-if="works.length" class="grid gap-2 sm:grid-cols-2">
      <AgentStepWork v-for="c in works" :key="agentCardKey(c)" :card="c" />
    </div>

    <AgentTagChips v-if="tags.length" :tags="tags" />

    <div v-if="entities.length" class="grid gap-1 sm:grid-cols-2">
      <component
        :is="item.previewable ? 'button' : 'div'"
        v-for="item in entities"
        :key="item.key"
        :type="item.previewable ? 'button' : undefined"
        :aria-label="item.previewable ? `预览「${item.title}」` : undefined"
        :class="
          cn(
            'group rounded-kun-md flex min-w-0 items-center gap-2 p-1 text-left',
            item.previewable && 'hover:bg-default-100 cursor-pointer'
          )
        "
        @click="item.previewable && preview.open(item.card)"
      >
        <div
          class="bg-default-100 rounded-kun-md relative aspect-[3/4] w-8 shrink-0 overflow-hidden"
        >
          <AgentCover :title="item.title" :cover="item.cover" />
        </div>
        <div class="min-w-0">
          <p
            :class="
              cn(
                'line-clamp-1 text-xs font-medium',
                item.previewable && 'group-hover:text-primary'
              )
            "
          >
            {{ item.title }}
          </p>
          <p
            v-if="item.caption"
            class="text-default-400 line-clamp-1 text-[11px]"
          >
            {{ item.caption }}
          </p>
        </div>
      </component>
    </div>

    <ul v-if="past.length" class="flex flex-col gap-1 text-xs">
      <li v-for="c in past" :key="c.id" class="flex min-w-0 items-center gap-2">
        <KunIcon
          name="lucide:message-square"
          class="text-default-400 size-3.5 shrink-0"
        />
        <NuxtLink
          :to="`/chat/${c.id}`"
          class="text-primary line-clamp-1 min-w-0 hover:underline"
        >
          {{ c.title }}
        </NuxtLink>
        <span v-if="c.updated_at" class="text-default-400 shrink-0">
          {{ formatDate(c.updated_at, { isShowYear: true }) }}
        </span>
      </li>
    </ul>

    <ul v-if="output.links.length" class="flex flex-col gap-0.5 text-xs">
      <li v-for="l in output.links" :key="l.url" class="min-w-0">
        <a
          :href="l.url"
          target="_blank"
          rel="noopener noreferrer nofollow"
          class="text-primary line-clamp-1 hover:underline"
        >
          {{ l.title || hostOf(l.url) }}
        </a>
      </li>
    </ul>
  </div>
</template>
