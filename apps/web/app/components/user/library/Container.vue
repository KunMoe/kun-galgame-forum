<script setup lang="ts">
import type {
  UserWorkRecord,
  UserWorkRecordList,
  WorkStateCounts
} from '#shared/utils/api/schemas'
import {
  KUN_GALGAME_LIBRARY_TABS,
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
  )
})

const state = computed(() => query.value.state)

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
  () => `user-library:${props.userId}:${state.value}:${stanceKey.value}`,
  async (client, cursor, { signal }) => {
    const res = await client.GET('/users/{user_id}/work-records', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          include_nsfw: allowsNsfw.value,
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

const totalOf = (counts: WorkStateCounts) =>
  counts.wish + counts.doing + counts.done + counts.on_hold + counts.dropped

const tabs = computed(() => {
  const counts = head.value?.state_counts
  return KUN_GALGAME_LIBRARY_TABS.map((tab) => {
    if (!counts) {
      return { value: tab.value, label: tab.label, icon: tab.icon }
    }
    const n = tab.value === 'all' ? totalOf(counts) : counts[tab.value]
    return { value: tab.value, label: `${tab.label} ${n}`, icon: tab.icon }
  })
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
    ? '还没有游玩记录, 在任意 Galgame 页面点「评分」即可标记游玩状态'
    : 'TA 还没有标记过游玩状态'
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

      <KunRadioGroup
        :model-value="state"
        :options="tabs"
        variant="pill"
        orientation="horizontal"
        color="primary"
        size="sm"
        @change="selectTab"
      />

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

      <KunButton
        v-if="hasMore"
        variant="flat"
        :loading="loadingMore"
        @click="loadMore"
      >
        加载更多
      </KunButton>
    </template>
  </div>
</template>
