<script setup lang="ts">
import type { SubscribedTopic } from '#shared/utils/api/schemas'

defineProps<{
  items: SubscribedTopic[]
  hasMore: boolean
  loadingMore: boolean
}>()

const emit = defineEmits<{ more: [] }>()

const topicLink = (item: SubscribedTopic) =>
  item.first_unread_floor
    ? `/topic/${item.id}?reply=${item.first_unread_floor}`
    : `/topic/${item.id}`
</script>

<template>
  <section
    v-if="items.length"
    class="border-default-200/60 mb-5 space-y-2 border-b pb-5"
  >
    <div class="flex items-center justify-between gap-2">
      <h3 class="flex items-center gap-1.5 text-sm font-medium">
        <KunIcon name="lucide:bell-ring" class="text-primary" />
        关注的话题有新回复
      </h3>
      <KunLink
        to="/message/follow"
        size="sm"
        color="default"
        underline="hover"
        class-name="text-default-500"
      >
        管理关注
      </KunLink>
    </div>

    <ul class="space-y-1">
      <li v-for="item in items" :key="item.id">
        <KunLink
          :to="topicLink(item)"
          color="default"
          underline="none"
          class-name="hover:bg-default-100 flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors"
        >
          <span class="min-w-0 flex-1 truncate">{{
            item.topic?.title ?? '话题'
          }}</span>
          <KunChip
            size="xs"
            color="primary"
            variant="flat"
            class-name="shrink-0"
          >
            {{ item.unread_reply_count }} 条新回复
          </KunChip>
          <span class="text-default-400 hidden shrink-0 text-xs sm:inline">
            <KunTime :time="item.last_reply_at" />
          </span>
        </KunLink>
      </li>
    </ul>

    <KunButton
      v-if="hasMore"
      variant="light"
      size="sm"
      :loading="loadingMore"
      @click="emit('more')"
    >
      查看更多
    </KunButton>
  </section>
</template>
