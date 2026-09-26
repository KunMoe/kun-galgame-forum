<script setup lang="ts">
import type { FollowingActivityItem } from '#shared/utils/api/schemas'

const props = defineProps<{ item: FollowingActivityItem }>()

const { isBlurred } = useContentStance()
const masked = computed(() => props.item.is_nsfw && isBlurred.value)
const coverFailed = ref(false)
const to = computed(() => props.item.in_site_path ?? props.item.url)
const external = computed(() => props.item.in_site_path === null)
</script>

<template>
  <div class="flex gap-3">
    <KunLink
      v-if="item.cover && !coverFailed"
      :to="to"
      :target="external ? '_blank' : undefined"
      class-name="shrink-0"
    >
      <div class="bg-default-100 aspect-[3/4] w-12 overflow-hidden rounded-md">
        <KunImage
          :src="item.cover.url"
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
      <p class="text-default-400 text-xs">
        <KunTime :time="item.occurred_at" />
      </p>
    </div>
  </div>
</template>
