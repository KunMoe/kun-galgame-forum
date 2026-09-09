<script setup lang="ts">
import {
  KUN_GALGAME_PLAY_STATE_DONE,
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'

const pageData = reactive({ page: 1, limit: 24 })

const { data, status } = await useKunFetch<PlaytimeMinePage>(
  '/galgame/playtime/mine',
  {
    query: computed(() => ({ ...pageData })),
    watch: [() => pageData.page]
  }
)

const summary = computed(() => {
  if (!data.value?.total) return ''
  const total = formatDurationMinutes(data.value.total_minutes)
  const parts = [`${data.value.total} 部作品`]
  if (total) parts.push(`合计 ${total}`)
  parts.push(`已通关 ${data.value.finished_works} 部`)
  return parts.join(' · ')
})

const statusColor = (value: string) => {
  if (
    value === 'done' ||
    (KUN_GALGAME_PLAY_STATE_DONE as readonly string[]).includes(value)
  ) {
    return 'success'
  }
  if (value === 'dropped') return 'danger'
  return 'default'
}
</script>

<template>
  <div class="space-y-3">
    <KunInfo
      color="info"
      title="只有你能看到这一页"
      description="逐条游玩记录是私密的, 站点只会公开「至少 3 位玩家上报, 且都已通关」的中位数。"
    />

    <p v-if="summary" class="text-default-600 text-sm">{{ summary }}</p>

    <KunInfo
      v-if="data?.truncated"
      color="warning"
      title="记录过多"
      description="你的记录超出了一次同步能取回的数量, 这里只展示其中最近改动的一部分。"
    />

    <div v-if="data && data.items.length" class="flex flex-col space-y-2">
      <KunCard
        v-for="item in data.items"
        :key="item.galgame.id"
        :href="`/galgame/${item.galgame.id}`"
        :is-transparent="false"
        content-class="flex items-center gap-3"
      >
        <KunImage
          :src="getEffectiveBanner(item.galgame, { variant: 'mini' })"
          loading="lazy"
          :alt="item.galgame.name"
          placeholder="/placeholder.webp"
          :thumbhash="resolveBannerThumbhash(item.galgame)"
          class="h-14 w-24 shrink-0 rounded-lg object-cover"
        />

        <div class="min-w-0 grow">
          <p class="line-clamp-1 font-medium">{{ item.galgame.name }}</p>
          <p class="text-default-500 line-clamp-1 text-sm">
            {{ item.galgame.name_original }}
          </p>
        </div>

        <div class="flex shrink-0 flex-col items-end gap-1">
          <span
            v-if="item.minutes > 0"
            class="font-medium tabular-nums"
          >
            {{ formatDurationMinutes(item.minutes) }}
          </span>
          <div v-if="item.status" class="flex items-center gap-1">
            <KunChip size="sm" variant="flat" :color="statusColor(item.status)">
              {{
                KUN_GALGAME_PLAY_STATE_MAP[
                  item.status as KunGalgamePlayStateRead
                ] ?? item.status
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

    <KunInfo
      v-else-if="!data && status !== 'pending'"
      color="danger"
      title="读取失败"
      description="暂时读不到你的游玩记录。如果你很久没有重新登录过, 请退出后重新登录以授予时长权限。"
    />

    <KunNull
      v-else-if="data"
      description="还没有游玩记录, 在任意 Galgame 页面点「标记游玩状态」即可记录"
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
