<script setup lang="ts">
import type { CollectionEntry } from '#shared/utils/api/schemas'

const props = defineProps<{
  entry: CollectionEntry
  mark?: { collected: boolean; playState: string | null }
  isCover?: boolean
}>()

const PLAYED = new Set(['done', 'done_one_route', 'done_main', 'done_all'])
const playChip = computed(() => {
  const state = props.mark?.playState
  if (!state) {
    return null
  }
  if (PLAYED.has(state)) {
    return { label: '已玩', color: 'success' as const }
  }
  if (state === 'doing') {
    return { label: '在玩', color: 'primary' as const }
  }
  if (state === 'wish') {
    return { label: '想玩', color: 'warning' as const }
  }
  return null
})

const nameOf = useWorkName()
const catalogName = useCatalogName()
const { isBlurred } = useContentStance()

const work = computed(() => props.entry.work_summary)
const href = computed(() => `/galgame/${work.value.id}`)
const name = computed(() => nameOf(work.value))
const originalName = computed(() =>
  work.value.display_name !== name.value ? work.value.display_name : ''
)
const masked = computed(() => work.value.is_nsfw && isBlurred.value)
const cover = computed(() =>
  masked.value ? work.value.cover_sfw : work.value.cover
)
const maker = computed(() =>
  work.value.maker ? catalogName(work.value.maker).name : ''
)
const releaseDate = computed(() => {
  const date = work.value.release_date
  if (!date) {
    return ''
  }
  switch (work.value.release_date_precision) {
    case 'year':
      return date.slice(0, 4)
    case 'month':
      return date.slice(0, 7)
    default:
      return date
  }
})
const staffLines = computed(() =>
  [
    { label: '剧本', people: props.entry.scenario_credits },
    { label: '原画', people: props.entry.illustration_credits }
  ].filter((line) => line.people.length)
)
</script>

<template>
  <div class="flex gap-4">
    <div class="relative w-24 shrink-0 self-start sm:w-32">
      <NuxtLink
        :to="href"
        :aria-label="name"
        tabindex="-1"
        class="bg-default-100 relative block overflow-hidden rounded-md"
      >
        <KunImage
          v-if="cover?.url"
          :src="cover.url"
          :alt="name"
          :thumbhash="cover.thumbhash ?? undefined"
          aspect-ratio="5 / 7"
          class-name="transition-transform duration-300 hover:scale-105"
        />
        <div
          v-else
          class="text-default-400 flex items-center justify-center"
          style="aspect-ratio: 5 / 7"
        >
          <KunIcon name="lucide:image-off" class="size-6" />
        </div>
      </NuxtLink>
      <div v-if="$slots.cover" class="absolute right-1.5 bottom-1.5">
        <slot name="cover" />
      </div>
    </div>

    <div class="flex min-w-0 flex-1 flex-col gap-2">
      <div class="flex items-start gap-2">
        <div class="min-w-0 flex-1">
          <h3 class="line-clamp-2 text-lg leading-snug font-semibold">
            <NuxtLink :to="href" class="hover:text-primary transition-colors">
              {{ name }}
            </NuxtLink>
          </h3>
          <div
            v-if="isCover || playChip || mark?.collected"
            class="mt-1 flex flex-wrap gap-1.5"
          >
            <KunChip v-if="isCover" color="primary" variant="flat" size="sm">
              封面
            </KunChip>
            <KunChip
              v-if="playChip"
              :color="playChip.color"
              variant="flat"
              size="sm"
            >
              {{ playChip.label }}
            </KunChip>
            <KunChip
              v-if="mark?.collected"
              color="default"
              variant="flat"
              size="sm"
            >
              已收藏
            </KunChip>
          </div>
          <p
            v-if="originalName"
            class="text-default-500 mt-0.5 line-clamp-1 text-sm"
            :title="originalName"
          >
            {{ originalName }}
          </p>
        </div>
        <div class="flex shrink-0 gap-1">
          <slot name="actions" />
        </div>
      </div>

      <div
        v-if="maker || releaseDate"
        class="text-default-600 flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1 text-sm"
      >
        <NuxtLink
          v-if="maker && work.maker"
          :to="`/galgame/official/${work.maker.id}`"
          class="hover:text-primary inline-flex min-w-0 items-center gap-1 leading-none transition-colors"
          :title="maker"
        >
          <KunIcon name="lucide:building-2" class="size-4 shrink-0" />
          <span class="truncate">{{ maker }}</span>
        </NuxtLink>
        <span
          v-if="releaseDate"
          class="inline-flex shrink-0 items-center gap-1 leading-none"
        >
          <KunIcon name="lucide:calendar" class="size-4 shrink-0" />
          <span class="tabular-nums">{{ releaseDate }}</span>
        </span>
      </div>

      <div v-if="staffLines.length" class="space-y-1 text-sm">
        <p v-for="line in staffLines" :key="line.label" class="flex gap-2">
          <span class="text-default-400 shrink-0">{{ line.label }}</span>
          <span class="min-w-0 truncate">
            <template v-for="(person, index) in line.people" :key="person.id">
              <span v-if="index" class="text-default-400"> / </span>
              <NuxtLink
                :to="`/galgame/staff/${person.id}`"
                class="text-default-700 hover:text-primary transition-colors"
              >
                {{ catalogName(person).name }}
              </NuxtLink>
            </template>
          </span>
        </p>
      </div>

      <div
        class="text-default-500 mt-auto flex items-center gap-4 pt-1 text-sm"
      >
        <span class="inline-flex items-center gap-1 leading-none" title="浏览">
          <KunIcon name="lucide:eye" class="size-4" />
          {{ formatNumber(work.view_count) }}
        </span>
        <span class="inline-flex items-center gap-1 leading-none" title="点赞">
          <KunIcon name="lucide:thumbs-up" class="size-4" />
          {{ formatNumber(work.like_count) }}
        </span>
        <span
          v-if="work.rating_score !== null"
          class="inline-flex items-center gap-1 leading-none"
          :title="`${work.rating_count} 个评分`"
        >
          <KunIcon name="lucide:star" class="size-4" />
          {{ work.rating_score.toFixed(1) }}
        </span>
        <div class="ml-auto">
          <slot name="mark" />
        </div>
      </div>
    </div>
  </div>
</template>
