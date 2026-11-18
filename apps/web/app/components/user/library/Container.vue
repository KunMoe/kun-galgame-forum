<script setup lang="ts">
import type {
  UserWorkRecord,
  UserWorkRecordList,
  WorkStateCounts
} from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_LIBRARY_SORTS,
  KUN_GALGAME_LIBRARY_TABS,
  type KunGalgameLibrarySort,
  type KunGalgameLibraryTab
} from '~/constants/galgame-playtime'

const props = defineProps<{
  userId: number
  isOwner: boolean
}>()

type LibraryItem = UserWorkRecord & { id: string }
type LibraryPage = Omit<UserWorkRecordList, 'items'> & {
  items: LibraryItem[]
}

const { allowsNsfw, stanceKey } = useContentStance()
// The field is named state for the ?state= URL, and useQueryState's own
// `state` (every field at once) would shadow a destructured ref of that name.
const { state: query, set } = useQueryState({
  state: queryEnum(
    KUN_GALGAME_LIBRARY_TABS.map((tab) => tab.value),
    'all'
  ),
  sort: queryEnum(
    KUN_GALGAME_LIBRARY_SORTS.map((sort) => sort.value),
    'updated_desc'
  )
})

const state = computed(() => query.value.state)
const sort = computed({
  get: () => query.value.sort,
  set: (value: KunGalgameLibrarySort) => set({ sort: value })
})
const sortOptions = KUN_GALGAME_LIBRARY_SORTS.map((option) => ({
  value: option.value,
  label: option.label
}))
const listKey = computed(() => `${state.value}:${sort.value}`)

const {
  items,
  head,
  hasMore,
  loadingMore,
  loadMore,
  problem,
  status,
  pending
} = useCursorList<LibraryItem, LibraryPage>(
  () => `user-library:${props.userId}:${listKey.value}:${stanceKey.value}`,
  async (client, cursor, { signal }) => {
    const res = await client.GET('/users/{user_id}/work-records', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          include_nsfw: allowsNsfw.value,
          record: 'any',
          sort: sort.value,
          ...(state.value === 'all' ? {} : { state: [state.value] }),
          ...(cursor ? { cursor } : {})
        }
      },
      signal
    })
    if (!res.data) {
      return res
    }
    const withIds = res.data.items.map((item) => ({
      ...item,
      id: item.work_summary.id
    }))
    return { ...res, data: { ...res.data, items: withIds } }
  }
)

const sentinel = useTemplateRef('sentinel')
const { loadNext } = useAutoLoadMore(sentinel, {
  hasMore,
  loading: loadingMore,
  loadMore,
  resetOn: listKey
})

const totalOf = (counts: WorkStateCounts) =>
  counts.wish +
  counts.doing +
  counts.done +
  counts.on_hold +
  counts.dropped +
  counts.none

const playtime = useUserPlaytimeTotal(() => String(props.userId))
const stats = computed(() => {
  const counts = head.value?.state_counts
  if (!counts || !totalOf(counts)) {
    return []
  }
  const compact = compactDurationMinutes(playtime.value)
  return [
    { label: '部作品', value: String(totalOf(counts)) },
    {
      label: '合计时长',
      value: compact?.value ?? '—',
      unit: compact?.unit,
      title: formatDurationMinutes(playtime.value) || undefined
    },
    { label: '部玩过', value: String(counts.done) }
  ]
})

const tabs = computed(() => {
  const counts = head.value?.state_counts
  return KUN_GALGAME_LIBRARY_TABS.map((tab) => {
    if (!counts) {
      return { value: tab.value, label: tab.label, icon: tab.icon }
    }
    const n = tab.value === 'all' ? totalOf(counts) : counts[tab.value]
    if (tab.value === 'none' && !n && state.value !== 'none') {
      return null
    }
    return { value: tab.value, label: `${tab.label} ${n}`, icon: tab.icon }
  }).filter((tab) => tab !== null)
})

const selectTab = (value: string) => {
  set({ state: value as KunGalgameLibraryTab })
}

const isPrivate = computed(() => problem.value?.code === 'PLAY_RECORDS_PRIVATE')

const galgames = useWorkCards(() =>
  items.value.map((item) => item.work_summary)
)
const recordOf = computed(
  () => new Map(items.value.map((item) => [Number(item.id), item]))
)

const privacy = useKunApi(
  () => `me-privacy-settings:${props.userId}`,
  (client, { signal }) => client.GET('/me/privacy-settings', { signal }),
  { immediate: props.isOwner }
)
const ownRecordsPublic = computed(
  () => privacy.data.value?.is_play_records_public ?? null
)

const settingHref = computed(() => `/user/${props.userId}/setting#play-records`)

const emptyText = computed(() => {
  if (state.value !== 'all') {
    return '这里还没有作品'
  }
  return props.isOwner
    ? '还没有游玩记录, 在任意 Galgame 页面点「评分」即可标记游玩状态和时长'
    : 'TA 还没有游玩记录'
})
</script>

<template>
  <div class="space-y-3">
    <KunNull v-if="isPrivate" description="TA 未公开游玩记录" />

    <template v-else>
      <KunSkeleton
        v-if="isOwner && privacy.pending.value"
        variant="text"
        width="14rem"
        class-name="text-sm"
      />
      <KunInfo
        v-else-if="isOwner && ownRecordsPublic === false"
        color="warning"
        title="仅自己可见"
      >
        你关闭了游玩记录公开, 别人打开这一页只会看到「未公开」。
        <KunLink :to="settingHref" size="sm" underline="always">
          去设置里打开
        </KunLink>
      </KunInfo>
      <p
        v-else-if="isOwner && ownRecordsPublic"
        class="text-default-500 text-sm"
      >
        你的游戏库对所有人公开。
        <KunLink :to="settingHref" size="sm" underline="hover">
          在设置里关闭
        </KunLink>
      </p>

      <UserLibrarySkeleton v-if="pending && !head" part="stats" />
      <div v-else-if="stats.length" class="grid grid-cols-3 gap-3">
        <div
          v-for="stat in stats"
          :key="stat.label"
          class="bg-default-100 min-w-0 rounded-lg px-3 py-3 sm:px-4"
          :title="stat.title"
        >
          <p
            class="text-xl font-bold whitespace-nowrap tabular-nums sm:text-2xl"
          >
            {{ stat.value
            }}<span
              v-if="stat.unit"
              class="text-default-500 ms-0.5 text-sm font-normal"
              >{{ stat.unit }}</span
            >
          </p>
          <p class="text-default-500 text-sm">{{ stat.label }}</p>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3">
        <KunRadioGroup
          :model-value="state"
          :options="tabs"
          variant="pill"
          orientation="horizontal"
          color="primary"
          size="sm"
          @change="selectTab"
        />
        <KunSelect
          v-model="sort"
          :options="sortOptions"
          size="sm"
          class-name="w-32"
          aria-label="排序"
        />
      </div>

      <p v-if="!allowsNsfw" class="text-default-500 text-sm">
        未开启 NSFW 显示，成人作品不在这里，也不计入数量
      </p>

      <KunInfo
        v-if="problem"
        color="danger"
        title="读取失败"
        description="暂时读不到游玩记录, 请稍后再试。"
      />

      <GalgameCard v-else-if="galgames.length" :galgames="galgames">
        <template #meta="{ galgame }">
          <UserLibraryMeta
            v-if="recordOf.get(galgame.id)"
            :record="recordOf.get(galgame.id)!"
          />
        </template>
      </GalgameCard>

      <GalgameCardGridSkeleton v-else-if="pending" />

      <KunNull v-else-if="status !== 'pending'" :description="emptyText" />

      <div v-if="items.length" ref="sentinel" class="flex justify-center">
        <KunButton
          v-if="hasMore"
          variant="flat"
          :loading="loadingMore"
          @click="loadNext"
        >
          加载更多
        </KunButton>
      </div>

      <p v-if="isOwner" class="text-default-500 text-sm">
        想让桌面客户端自动记录游玩时长?
        <KunLink
          to="https://developer.nextmoe.dev/docs/playtime"
          target="_blank"
          underline="hover"
          size="sm"
        >
          在 NextMoe 开发者平台创建一个应用
        </KunLink>
        即可, playtime 权限无需审批, 它上报的记录会和这里合并显示。
      </p>
    </template>
  </div>
</template>
