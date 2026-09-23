<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import type { Activity } from '#shared/utils/api/schemas'
import { KUN_ACTIVITY_TYPE_TYPE } from '~/constants/activity'
import { activitySummaryText } from '~/utils/activity'
import { toKunUser } from '~/utils/userRef'

const settings = usePersistSettingsStore()
const { allowsNsfw } = useContentStance()
const nameOf = useWorkName()

const query = computed(() => ({
  include_nsfw: allowsNsfw.value,
  include_galgames_without_resources: settings.showKUNGalgameNoResource,
  limit: 50
}))

const { items, status, hasMore, loadingMore, loadMore } =
  await useCursorList<Activity>(
    () => `activities:${JSON.stringify(query.value)}`,
    (api, cursor, { signal }) =>
      api.GET('/activities', {
        params: { query: { ...query.value, ...(cursor ? { cursor } : {}) } },
        signal
      })
  )

const sentinel = ref<HTMLElement | null>(null)
useIntersectionObserver(
  sentinel,
  ([entry]) => {
    if (entry?.isIntersecting) loadMore()
  },
  { rootMargin: '400px' }
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="动态时间线"
      description="动态时间线, 展示全站 话题, 回复, Galgame 与社区的最新 Galgame 资源, Galgame 动态, Galgame 讨论, Galgame 评论等"
    />

    <KunNull
      v-if="status !== 'pending' && !items.length"
      description="暂无动态"
    />

    <div v-else class="relative space-y-6">
      <div class="bg-primary/20 absolute top-6 bottom-0 left-4 w-0.5" />

      <div
        v-for="activity in items"
        :key="activity.id"
        class="flex items-center gap-3"
      >
        <KunAvatar
          v-if="activity.performer"
          :user="toKunUser(activity.performer)"
        />

        <div class="flex flex-col space-y-2">
          <KunLink
            underline="none"
            color="default"
            :to="activity.path"
            class-name="hover:text-primary block space-x-3 break-all transition-colors"
          >
            <KunText
              class-name="whitespace-normal!"
              :content="activitySummaryText(activity, nameOf)"
            />
            <KunChip color="primary" size="xs">
              {{ KUN_ACTIVITY_TYPE_TYPE[activity.activity_type] }}
            </KunChip>
          </KunLink>

          <div class="flex items-center space-x-2">
            <span class="text-default-500 text-sm">
              <template v-if="activity.performer"
                >{{ toKunUser(activity.performer).name }} 发布于 </template
              ><KunTime :time="activity.occurred_at" />
            </span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="items.length" ref="sentinel" class="flex justify-center pt-1">
      <KunButton
        v-if="hasMore"
        variant="light"
        :loading="loadingMore"
        @click="loadMore"
      >
        加载更多
      </KunButton>
      <span v-else class="text-default-400 text-sm">没有更多动态了</span>
    </div>
  </div>
</template>
