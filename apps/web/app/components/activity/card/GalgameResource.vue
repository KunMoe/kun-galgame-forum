<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_USER_TEXT_CHIP_CLASS
} from '~/constants/galgame'

const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as GalgameActivityData | undefined
)
const resource = computed(() => data.value?.resource)
const gid = computed(() => data.value?.galgame_id ?? 0)
const galgameLink = computed(() =>
  gid.value ? `/galgame/${gid.value}` : props.activity.link
)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="flex gap-3">
      <KunLink :to="galgameLink" class-name="shrink-0">
        <div
          class="bg-default-100 aspect-video w-32 overflow-hidden rounded-lg sm:w-44"
        >
          <img
            v-if="data?.cover_hash"
            :src="imageHashUrl(imageCdnBase(), data.cover_hash, 'mini')"
            :alt="data?.name"
            loading="lazy"
            class="h-full w-full object-cover"
          />
        </div>
      </KunLink>

      <div class="min-w-0 flex-1 space-y-1.5">
        <KunLink
          underline="none"
          color="default"
          :to="galgameLink"
          class-name="hover:text-primary block"
        >
          <h3 class="line-clamp-1 font-medium break-all">{{ data?.name }}</h3>
        </KunLink>

        <div v-if="resource" class="flex flex-wrap items-center gap-1.5">
          <KunChip
            v-if="resource.type"
            size="sm"
            variant="flat"
            color="primary"
          >
            {{ KUN_GALGAME_RESOURCE_TYPE_MAP[resource.type] ?? resource.type }}
          </KunChip>
          <KunChip
            v-if="resource.platform"
            size="sm"
            variant="flat"
            color="secondary"
          >
            {{
              KUN_GALGAME_RESOURCE_PLATFORM_MAP[resource.platform] ??
              resource.platform
            }}
          </KunChip>
          <KunChip
            v-if="resource.language"
            size="sm"
            variant="flat"
            color="success"
          >
            {{
              KUN_GALGAME_RESOURCE_LANGUAGE_MAP[resource.language] ??
              resource.language
            }}
          </KunChip>
          <KunChip v-if="resource.size" size="sm" variant="flat" :class-name="KUN_USER_TEXT_CHIP_CLASS">
            {{ resource.size }}
          </KunChip>
        </div>

        <p
          v-if="resource?.note"
          class="text-default-500 line-clamp-3 text-sm break-all whitespace-pre-wrap"
        >
          {{ resource.note }}
        </p>

        <span
          v-if="resource?.like_count"
          class="text-default-500 flex items-center gap-1 text-sm"
        >
          <KunIcon name="lucide:thumbs-up" class="size-3.5" />
          {{ resource.like_count }}
        </span>
      </div>
    </div>
  </ActivityCardShell>
</template>
