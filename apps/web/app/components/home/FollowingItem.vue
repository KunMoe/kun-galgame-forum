<script setup lang="ts">
import type { FollowingActivityItem } from '#shared/utils/api/schemas'
import { workReleaseText } from '~/utils/galgame/releaseDate'

const props = defineProps<{ item: FollowingActivityItem }>()

const { isBlurred } = useContentStance()
const nameOf = useCatalogName()
const masked = computed(
  () => (props.item.is_nsfw || !!props.item.work_preview?.is_nsfw) && isBlurred.value
)
const coverFailed = ref(false)
const work = computed(() => props.item.work_preview)
const cover = computed(() => props.item.cover ?? work.value?.cover ?? null)
const release = computed(() =>
  work.value?.release_date
    ? workReleaseText(
        work.value.release_date,
        work.value.release_date_precision
      )
    : ''
)
const to = computed(() => props.item.in_site_path ?? props.item.url)
const external = computed(() => props.item.in_site_path === null)
</script>

<template>
  <div class="flex gap-3">
    <KunLink
      v-if="cover && !coverFailed"
      :to="to"
      :target="external ? '_blank' : undefined"
      class-name="shrink-0"
    >
      <div
        :class="
          cn(
            'bg-default-100 aspect-[3/4] overflow-hidden rounded-md',
            work ? 'w-20 sm:w-24' : 'w-12'
          )
        "
      >
        <KunImage
          :src="cover.url"
          :alt="item.title"
          loading="lazy"
          aspect-ratio="3 / 4"
          :image-class-name="masked ? 'blur-xl' : undefined"
          @error="coverFailed = true"
        />
      </div>
    </KunLink>

    <div class="min-w-0 flex-1 space-y-1">
      <div class="flex items-center gap-1.5">
        <KunLink
          :to="to"
          :target="external ? '_blank' : undefined"
          underline="none"
          color="default"
          class-name="text-default-800 hover:text-primary min-w-0 truncate text-sm font-medium"
        >
          {{ item.title }}
        </KunLink>
        <KunIcon
          v-if="external"
          name="lucide:external-link"
          class="text-default-400 shrink-0 text-xs"
        />
        <KunChip v-if="item.is_nsfw" size="xs" color="danger" variant="flat">
          NSFW
        </KunChip>
      </div>
      <p
        v-if="work?.maker || release"
        class="text-default-500 flex flex-wrap items-center gap-x-2 text-xs"
      >
        <KunLink
          v-if="work?.maker"
          :to="`/galgame/official/${work.maker.id}`"
          underline="none"
          color="default"
          class-name="text-default-500 hover:text-primary"
        >
          {{ nameOf(work.maker).name }}
        </KunLink>
        <span v-if="release" class="flex items-center gap-1">
          <KunIcon name="lucide:calendar" class="size-3" />
          {{ release }}
        </span>
      </p>
      <div v-if="work?.top_tags.length" class="flex flex-wrap gap-1">
        <NuxtLink
          v-for="tag in work.top_tags"
          :key="tag.id"
          :to="`/galgame/tag/${tag.id}`"
          class="bg-default-100 text-default-600 hover:bg-primary/15 hover:text-primary rounded-md px-1.5 py-0.5 text-xs transition-colors"
        >
          {{ nameOf(tag).name }}
        </NuxtLink>
      </div>
      <p
        v-if="item.excerpt"
        :class="
          cn(
            'text-default-500 line-clamp-2 text-sm break-words',
            masked && 'blur-sm select-none'
          )
        "
      >
        {{ item.excerpt }}
      </p>
      <p class="text-default-400 flex items-center gap-2 text-xs">
        <template v-if="item.topic_times">
          <KunTime :time="item.topic_times.bumped_at" />
          <span
            v-if="item.topic_times.edited_at"
            class="flex items-center gap-1"
          >
            <KunIcon name="lucide:pencil" class="size-3" />
            <KunTime :time="item.topic_times.edited_at" />
          </span>
        </template>
        <KunTime v-else :time="item.occurred_at" />
      </p>
    </div>
  </div>
</template>
