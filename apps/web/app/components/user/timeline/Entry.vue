<script setup lang="ts">
import type { UserActivityGroup } from '#shared/utils/api/schemas'
import { followingGroupPhrase, followingSiteLabel } from '~/utils/following'
import { timelineVerbIcon } from '~/utils/userTimeline'

const props = defineProps<{
  group: UserActivityGroup
  includeNsfw: boolean
}>()

const emit = defineEmits<{
  gone: [id: string]
}>()

const { items, hasHidden, nextCursor, loading, loadItems } =
  useActivityGroupItems(
    () => props.group,
    () => props.includeNsfw,
    () => emit('gone', props.group.id)
  )
const phrase = computed(() => followingGroupPhrase(props.group))
const siteLabel = computed(() => followingSiteLabel(props.group.site))
</script>

<template>
  <article class="relative pl-11">
    <span
      class="bg-content1 border-default-200 text-primary absolute top-0 left-0 flex size-8 items-center justify-center rounded-full border"
      aria-hidden="true"
    >
      <KunIcon :name="timelineVerbIcon(group.verb)" class="size-4" />
    </span>

    <header class="flex min-h-8 flex-wrap items-center gap-2 text-sm">
      <span class="text-default-700 font-medium">{{ phrase }}</span>
      <KunChip v-if="siteLabel" size="xs" variant="flat">
        {{ siteLabel }}
      </KunChip>
      <span class="text-default-400 ml-auto text-xs">
        <KunTime :time="group.latest_at" />
      </span>
    </header>

    <div class="mt-2 space-y-4">
      <HomeFollowingItem v-for="item in items" :key="item.id" :item="item" />

      <KunButton
        v-if="hasHidden || nextCursor"
        variant="light"
        size="sm"
        :loading="loading"
        @click="loadItems"
      >
        {{ hasHidden ? `查看全部 ${group.item_count} 条` : '加载更多' }}
      </KunButton>
    </div>
  </article>
</template>
