<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
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
const api = useApiClient()

const query = computed(() => ({
  include_nsfw: allowsNsfw.value,
  limit: LIMIT,
  ...(verb.value === 'all' ? {} : { verbs: [verb.value] })
}))

const { data, status, problem, pending } = useKunApi(
  () => `user-activities:${props.userId}:${verb.value}:${allowsNsfw.value}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/activities', {
      params: { path: { user_id: String(props.userId) }, query: query.value },
      signal
    })
)

const more = ref<UserActivityGroup[]>([])
const moreCursor = ref<string>()
const loadedMore = ref(false)
const loadingMore = ref(false)
const gone = ref(new Set<string>())

watch(
  data,
  () => {
    more.value = []
    moreCursor.value = undefined
    loadedMore.value = false
  },
  { flush: 'sync' }
)

const groups = computed(() =>
  [...(data.value?.items ?? []), ...more.value].filter(
    (group) => !gone.value.has(group.id)
  )
)
const days = computed(() => groupTimelineByDay(groups.value))
const newestYear = computed(
  () => groups.value[0]?.calendar_date.slice(0, 4) ?? ''
)
const nextCursor = computed(() =>
  loadedMore.value ? moreCursor.value : data.value?.next_cursor
)

const loadMore = async () => {
  const cursor = nextCursor.value
  if (!cursor || loadingMore.value) {
    return
  }
  const page = data.value
  loadingMore.value = true
  const result = await settle(
    api.GET('/users/{user_id}/activities', {
      params: {
        path: { user_id: String(props.userId) },
        query: { ...query.value, cursor }
      }
    })
  )
  loadingMore.value = false
  if (data.value !== page) {
    return
  }
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  more.value = [...more.value, ...result.data.items]
  moreCursor.value = result.data.next_cursor
  loadedMore.value = true
}

const markGone = (id: string) => {
  gone.value = new Set(gone.value).add(id)
}
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
      v-else-if="problem && !data"
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
            class="before:bg-default-200 relative space-y-3 before:absolute before:top-4 before:bottom-4 before:left-4 before:w-px"
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

        <KunButton
          v-if="nextCursor"
          variant="flat"
          class-name="w-full"
          :loading="loadingMore"
          @click="loadMore"
        >
          加载更多
        </KunButton>
      </div>
    </KunLoading>
  </div>
</template>
