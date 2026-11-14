<script setup lang="ts">
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  kunPlayStateColor,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import type { WorkPlaytimeList } from '#shared/utils/api/schemas'

type Item = WorkPlaytimeList['items'][number]

const { allowsNsfw, stanceKey, isBlurred } = useContentStance()
const { id: myId } = storeToRefs(usePersistUserStore())
const nameOf = useCatalogName()
const pageData = reactive({ page: usePageQuery(), limit: 24 })

const { data, status, problem, pending } = useKunApi<WorkPlaytimeList>(
  () => `me-playtimes:${pageData.page}:${pageData.limit}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/me/playtimes', {
      params: {
        query: {
          page: pageData.page,
          limit: pageData.limit,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    })
)

const stats = computed(() =>
  data.value?.total
    ? [
        { label: '部作品', value: String(data.value.total) },
        {
          label: '合计时长',
          value: formatDurationMinutes(data.value.total_minutes) || '—'
        },
        { label: '部已通关', value: String(data.value.finished_work_count) }
      ]
    : []
)

const coverOf = (item: Item) =>
  item.work_summary.is_nsfw && isBlurred.value
    ? item.work_summary.cover_sfw
    : item.work_summary.cover
const stateLabel = (state: string) =>
  KUN_GALGAME_PLAY_STATE_MAP[state as KunGalgamePlayStateRead] ?? state

const { paginationPage, isPaginationPending } = usePaginationScroll(
  toRef(pageData, 'page'),
  () => status.value === 'pending'
)
</script>

<template>
  <div class="space-y-6">
    <div v-if="stats.length" class="grid grid-cols-3 gap-3">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="bg-default-100 rounded-lg px-4 py-3"
      >
        <p class="text-2xl font-bold tabular-nums">{{ stat.value }}</p>
        <p class="text-default-500 text-sm">{{ stat.label }}</p>
      </div>
    </div>

    <div class="text-default-500 space-y-1 text-sm">
      <p>
        游玩状态和时长默认显示在主页的「游戏库」里, 所有人都能看到,
        <KunLink
          :to="`/user/${myId}/setting#play-records`"
          size="sm"
          underline="always"
        >
          在设置里关闭
        </KunLink>
        后只有你自己能看到。
      </p>
      <p v-if="data?.is_truncated">部分记录未能读取，统计可能偏少。</p>
      <p v-if="data && !allowsNsfw">
        未开启 NSFW 显示，成人作品的记录不在这里，也不计入统计。
      </p>
    </div>

    <div
      ref="paginationStart"
      class="pagination-results flex scroll-mt-20 flex-col gap-6"
    >
      <template v-if="data && data.items.length">
        <KunLoading :loading="isPaginationPending">
          <ul class="grid gap-x-6 gap-y-1 md:grid-cols-2">
            <li v-for="item in data.items" :key="item.work_summary.id">
              <NuxtLink
                :to="`/galgame/${item.work_summary.id}`"
                class="hover:bg-default-100 flex items-center gap-3 rounded-lg p-2 transition-colors"
              >
                <div
                  class="bg-default-100 w-14 shrink-0 overflow-hidden rounded-md"
                >
                  <KunImage
                    v-if="coverOf(item)?.url"
                    :src="coverOf(item)!.url"
                    :alt="nameOf(item.work_summary).name"
                    :thumbhash="coverOf(item)?.thumbhash ?? undefined"
                    aspect-ratio="5 / 7"
                    loading="lazy"
                  />
                  <div
                    v-else
                    class="text-default-400 flex items-center justify-center"
                    style="aspect-ratio: 5 / 7"
                  >
                    <KunIcon name="lucide:image-off" class="size-5" />
                  </div>
                </div>

                <div class="min-w-0 flex-1 space-y-1">
                  <p class="line-clamp-1 font-medium">
                    {{ nameOf(item.work_summary).name }}
                  </p>
                  <p
                    v-if="nameOf(item.work_summary).original"
                    class="text-default-500 line-clamp-1 text-sm"
                  >
                    {{ nameOf(item.work_summary).original }}
                  </p>
                  <KunChip
                    v-if="item.play_state"
                    size="xs"
                    variant="flat"
                    :color="kunPlayStateColor(item.play_state)"
                  >
                    {{ stateLabel(item.play_state) }}
                  </KunChip>
                </div>

                <div class="shrink-0 text-right">
                  <p
                    v-if="item.minutes > 0"
                    class="text-lg font-semibold tabular-nums"
                  >
                    {{ formatDurationMinutes(item.minutes) }}
                  </p>
                  <p v-else class="text-default-400 text-sm">未记录时长</p>
                  <p
                    v-if="item.client_count > 1"
                    class="text-default-400 text-xs"
                  >
                    {{ item.client_count }} 个应用上报
                  </p>
                </div>
              </NuxtLink>
            </li>
          </ul>
        </KunLoading>

        <KunPagination
          v-if="data.total > pageData.limit"
          v-model:current-page="paginationPage"
          :total-page="Math.ceil(data.total / pageData.limit)"
          :is-loading="status === 'pending'"
        />
      </template>

      <UserListSkeleton v-else-if="pending" shape="media" />
      <KunInfo
        v-else-if="problem && status !== 'pending'"
        color="danger"
        title="读取失败"
        description="暂时读不到你的游玩记录。"
      />
      <KunNull
        v-else-if="data"
        description="还没有游玩记录, 在任意 Galgame 页面点「标记」或「评分」即可记录游玩状态和时长"
      />
    </div>

    <p class="text-default-500 text-sm">
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
  </div>
</template>
