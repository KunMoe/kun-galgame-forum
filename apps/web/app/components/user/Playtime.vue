<script setup lang="ts">
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  kunPlayStateColor,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import type { WorkPlaytimeList } from '#shared/utils/api/schemas'

const { allowsNsfw, stanceKey } = useContentStance()
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

const summary = computed(() => {
  if (!data.value?.total) return ''
  const total = formatDurationMinutes(data.value.total_minutes)
  const parts = [`${data.value.total} 部作品`]
  if (total) parts.push(`合计 ${total}`)
  parts.push(`已通关 ${data.value.finished_work_count} 部`)
  return parts.join(' · ')
})

const namesOf = (item: WorkPlaytimeList['items'][number]) =>
  nameOf(item.work_summary)

const bannerOf = (item: WorkPlaytimeList['items'][number]) =>
  item.work_summary.banner ?? item.work_summary.cover
</script>

<template>
  <div class="space-y-3">
    <KunInfo color="info" title="游玩记录默认公开">
      你标记的游玩状态和上报的时长会显示在主页的「游戏库」里, 所有人都能看到,
      和评分一样。
      <KunLink
        :to="`/user/${myId}/setting#play-records`"
        size="sm"
        underline="always"
      >
        在设置里关闭
      </KunLink>
      后只有你自己能看到。
    </KunInfo>

    <p v-if="summary" class="text-default-600 text-sm">{{ summary }}</p>
    <p v-if="data?.is_truncated" class="text-default-500 text-sm">
      部分记录未能读取，统计可能偏少
    </p>
    <p v-if="data && !allowsNsfw" class="text-default-500 text-sm">
      未开启 NSFW 显示，成人作品的记录不在这里，也不计入统计
    </p>

    <div v-if="data && data.items.length" class="flex flex-col space-y-2">
      <KunCard
        v-for="item in data.items"
        :key="item.work_summary.id"
        :href="`/galgame/${item.work_summary.id}`"
        :is-transparent="false"
        content-class="flex items-center gap-3"
      >
        <KunImage
          :src="bannerOf(item)?.url ?? '/placeholder.webp'"
          loading="lazy"
          :alt="namesOf(item).name"
          placeholder="/placeholder.webp"
          :thumbhash="bannerOf(item)?.thumbhash ?? undefined"
          class="h-14 w-24 shrink-0 rounded-lg object-cover"
        />

        <div class="min-w-0 grow">
          <p class="line-clamp-1 font-medium">{{ namesOf(item).name }}</p>
          <p class="text-default-500 line-clamp-1 text-sm">
            {{ namesOf(item).original }}
          </p>
        </div>

        <div class="flex shrink-0 flex-col items-end gap-1">
          <span v-if="item.minutes > 0" class="font-medium tabular-nums">
            {{ formatDurationMinutes(item.minutes) }}
          </span>
          <div v-if="item.play_state" class="flex items-center gap-1">
            <KunChip
              size="sm"
              variant="flat"
              :color="kunPlayStateColor(item.play_state)"
            >
              {{
                KUN_GALGAME_PLAY_STATE_MAP[
                  item.play_state as KunGalgamePlayStateRead
                ] ?? item.play_state
              }}
            </KunChip>
          </div>
        </div>
      </KunCard>

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <UserListSkeleton v-else-if="pending" shape="media" />
    <KunInfo
      v-else-if="problem && status !== 'pending'"
      color="danger"
      title="读取失败"
      description="暂时读不到你的游玩记录。"
    />

    <KunNull
      v-else-if="data"
      description="还没有游玩记录, 在任意 Galgame 页面点「评分」即可记录游玩状态和时长"
    />

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
