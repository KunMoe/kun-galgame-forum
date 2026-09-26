<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type {
  FollowingActivityGroup,
  FollowingActivityItem
} from '#shared/utils/api/schemas'
import { followingGroupPhrase, followingSiteLabel } from '~/utils/following'

const props = defineProps<{
  group: FollowingActivityGroup
  includeNsfw: boolean
}>()

const api = useApiClient()

const expanded = ref<FollowingActivityItem[] | null>(null)
const nextCursor = ref<string | undefined>()
const loading = ref(false)

const items = computed(() => expanded.value ?? props.group.items)
const hasHidden = computed(
  () =>
    expanded.value === null && props.group.item_count > props.group.items.length
)
const phrase = computed(() => followingGroupPhrase(props.group))
const siteLabel = computed(() => followingSiteLabel(props.group.site))

const loadItems = async () => {
  if (loading.value) {
    return
  }
  loading.value = true
  const result = await settle(
    api.GET('/activity-groups/{group_id}/items', {
      params: {
        path: { group_id: props.group.id },
        query: {
          include_nsfw: props.includeNsfw,
          limit: 20,
          ...(nextCursor.value ? { cursor: nextCursor.value } : {})
        }
      }
    })
  )
  loading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  expanded.value = [...(expanded.value ?? []), ...result.data.items]
  nextCursor.value = result.data.next_cursor
}
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
