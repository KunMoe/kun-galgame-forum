<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_USER_TEXT_CHIP_CLASS
} from '~/constants/galgame'
import type {
  Activity,
  ActivityResource,
  WorkRef
} from '#shared/utils/api/schemas'

defineProps<{
  activity: Activity
  work: WorkRef
  resource: ActivityResource
}>()

const nameOf = useWorkName()
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="flex gap-3">
      <ActivityCardWorkCover :work="work" />

      <div class="min-w-0 flex-1 space-y-1.5">
        <KunLink
          underline="none"
          color="default"
          :to="`/galgame/${work.id}`"
          class-name="hover:text-primary block"
        >
          <h3 class="line-clamp-1 font-medium break-all">
            {{ nameOf(work) }}
          </h3>
        </KunLink>

        <div class="flex flex-wrap items-center gap-1.5">
          <KunChip size="sm" variant="flat" color="primary">
            {{
              KUN_GALGAME_RESOURCE_TYPE_MAP[resource.resource_type] ??
              resource.resource_type
            }}
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
          <KunChip
            v-if="resource.size"
            size="sm"
            variant="flat"
            :class-name="KUN_USER_TEXT_CHIP_CLASS"
          >
            {{ resource.size }}
          </KunChip>
        </div>

        <p
          v-if="resource.note"
          class="text-default-500 line-clamp-3 text-sm break-all whitespace-pre-wrap"
        >
          {{ resource.note }}
        </p>

        <span
          v-if="resource.like_count"
          class="text-default-500 flex items-center gap-1 text-sm"
        >
          <KunIcon name="lucide:thumbs-up" class="size-3.5" />
          {{ resource.like_count }}
        </span>
      </div>
    </div>
  </ActivityCardShell>
</template>
