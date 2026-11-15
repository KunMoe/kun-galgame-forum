<script setup lang="ts">
import type { UserActivityGroup } from '#shared/utils/api/schemas'
import {
  TIMELINE_VERBS,
  groupTimelineByDay,
  timelineDayLabel,
  timelineFilterTabs
} from '~/utils/userTimeline'

const props = defineProps<{
  userId: number
}>()

const LIMIT = 20

const { verb } = useQueryState({ verb: queryEnum(TIMELINE_VERBS, 'all') })
const { allowsNsfw } = useContentStance()

const { items, status, problem, pending, hasMore, loadingMore, loadMore } =
  useCursorList<UserActivityGroup>(
    () => `user-activities:${props.userId}:${verb.value}:${allowsNsfw.value}`,
    (client, cursor, { signal }) =>
      client.GET('/users/{user_id}/activities', {
        params: {
          path: { user_id: String(props.userId) },
          query: {
            include_nsfw: allowsNsfw.value,
            limit: LIMIT,
            ...(verb.value === 'all' ? {} : { verbs: [verb.value] }),
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )

const gone = ref(new Set<string>())
const groups = computed(() =>
  items.value.filter((group) => !gone.value.has(group.id))
)
const days = computed(() => groupTimelineByDay(groups.value))
const newestYear = computed(
  () => groups.value[0]?.calendar_date.slice(0, 4) ?? ''
)

const markGone = (id: string) => {
  gone.value = new Set(gone.value).add(id)
}

const sentinel = useTemplateRef('sentinel')
const { loadNext } = useAutoLoadMore(sentinel, {
  hasMore,
  loading: loadingMore,
  loadMore,
  resetOn: verb
})
</script>

<template>
  <div class="space-y-4">
    <KunTab
      v-model="verb"
      :items="timelineFilterTabs()"
      variant="light"
      size="sm"
      scrollable
    />

    <UserListSkeleton v-if="pending" shape="media" />

    <KunInfo
      v-else-if="problem && !items.length"
      color="danger"
      title="读取失败"
      description="动态暂时读不到，请稍后再试。"
    />

    <KunNull
      v-else-if="!groups.length"
      :description="verb === 'all' ? '还没有动态' : '没有这类动态'"
    />

    <KunLoading v-else :loading="status === 'pending'">
      <div class="space-y-6">
        <section
          v-for="day in days"
          :key="`${day.date}:${day.groups[0]!.id}`"
          class="space-y-3"
        >
          <h3 class="text-default-500 text-sm font-medium">
            {{ timelineDayLabel(day.date, newestYear) }}
          </h3>
          <ol
            class="before:bg-default-200 relative space-y-6 before:absolute before:top-4 before:bottom-4 before:left-4 before:w-px"
          >
            <li v-for="group in day.groups" :key="group.id">
              <UserTimelineEntry
                :group="group"
                :include-nsfw="allowsNsfw"
                @gone="markGone"
              />
            </li>
          </ol>
        </section>

        <div ref="sentinel" class="flex justify-center">
          <KunButton
            v-if="hasMore"
            variant="light"
            :loading="loadingMore"
            @click="loadNext"
          >
            加载更多
          </KunButton>
          <span v-else class="text-default-400 text-sm">没有更多动态了</span>
        </div>
      </div>
    </KunLoading>
  </div>
</template>
