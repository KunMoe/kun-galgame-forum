<script setup lang="ts">
import type { FollowingActivityGroup } from '#shared/utils/api/schemas'
import { followingGroupPhrase, followingSiteLabel } from '~/utils/following'

const props = defineProps<{
  group: FollowingActivityGroup
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
  <ActivityCardShell :performer="group.actor" :occurred-at="group.latest_at">
    <template #meta>
      <span class="text-default-600">{{ phrase }}</span>
      <KunChip v-if="siteLabel" size="xs" variant="flat">
        {{ siteLabel }}
      </KunChip>
    </template>

    <div class="space-y-3">
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
  </ActivityCardShell>
</template>
