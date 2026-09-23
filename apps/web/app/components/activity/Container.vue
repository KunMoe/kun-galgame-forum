<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import type { Activity, ActivityType } from '#shared/utils/api/schemas'
import {
  KUN_ACTIVITY_TYPE_TYPE,
  KUN_ACTIVITY_GROUPS,
  KUN_ACTIVITY_ICON_MAP
} from '~/constants/activity'
import { activitySummaryText } from '~/utils/activity'
import { toKunUser } from '~/utils/userRef'

const selectedType = ref<ActivityType>('topic_creation')

const settings = usePersistSettingsStore()
const { allowsNsfw } = useContentStance()
const nameOf = useWorkName()

const query = computed(() => ({
  activity_types: [selectedType.value],
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

const categoryPopover = ref<{ close: () => void } | null>(null)
const selectCategory = (type: ActivityType) => {
  selectedType.value = type
  categoryPopover.value?.close()
}

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
      name="最新动态"
      description="这里展示了论坛的所有动态, 包括 Galgame, Galgame 资源, Galgame 网站, 话题, 回复, 评论, 网站更新 等"
    />

    <KunPopover
      ref="categoryPopover"
      position="bottom-start"
      inner-class="w-60 max-h-[70vh] overflow-y-auto p-1.5"
    >
      <template #trigger>
        <div
          class="border-default-200 hover:border-primary flex w-fit cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors"
        >
          <KunIcon
            :name="KUN_ACTIVITY_ICON_MAP[selectedType]"
            class="text-primary"
          />
          <span class="font-medium">
            {{ KUN_ACTIVITY_TYPE_TYPE[selectedType] }}
          </span>
          <KunIcon name="lucide:chevron-down" class="text-default-400 ml-1" />
        </div>
      </template>

      <div
        v-for="group in KUN_ACTIVITY_GROUPS"
        :key="group.label"
        class="mb-1 last:mb-0"
      >
        <div class="text-default-400 px-2 py-1 text-xs font-medium">
          {{ group.label }}
        </div>
        <button
          v-for="type in group.types"
          :key="type"
          type="button"
          @click="selectCategory(type)"
          :class="
            cn(
              'flex w-full cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors',
              type === selectedType
                ? 'bg-primary/10 text-primary'
                : 'text-foreground hover:bg-default-100'
            )
          "
        >
          <KunIcon :name="KUN_ACTIVITY_ICON_MAP[type]" class="shrink-0" />
          <span class="flex-1 text-left">
            {{ KUN_ACTIVITY_TYPE_TYPE[type] }}
          </span>
          <KunIcon
            v-if="type === selectedType"
            name="lucide:check"
            class="text-primary shrink-0"
          />
        </button>
      </div>
    </KunPopover>

    <KunNull
      v-if="status !== 'pending' && !items.length"
      description="暂无动态"
    />

    <div
      v-for="activity in items"
      :key="activity.id"
      class="flex flex-col space-y-2"
    >
      <KunLink
        underline="none"
        color="default"
        :to="activity.path"
        class-name="hover:text-primary line-clamp-3 break-all transition-colors"
      >
        {{ activitySummaryText(activity, nameOf) }}
        <KunChip class-name="cursor-pointer" color="primary" size="xs">
          {{ KUN_ACTIVITY_TYPE_TYPE[activity.activity_type] }}
        </KunChip>
      </KunLink>

      <div class="flex items-center space-x-2">
        <KunUserChip
          v-if="activity.performer"
          size="sm"
          :user="toKunUser(activity.performer)"
        />
        <span class="text-default-500 text-sm">
          <KunTime :time="activity.occurred_at" />
        </span>
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
