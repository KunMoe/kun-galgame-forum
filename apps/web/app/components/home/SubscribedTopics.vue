<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { SubscribedTopic } from '#shared/utils/api/schemas'

const PAGE_SIZE = 5

const api = useApiClient()
const { allowsNsfw } = useContentStance()

const items = ref<SubscribedTopic[]>([])
const nextCursor = ref<string | undefined>()
const loading = ref(false)

const load = async (cursor?: string) => {
  loading.value = true
  const result = await settle(
    api.GET('/me/topic-subscriptions', {
      params: {
        query: {
          has_unread: true,
          include_nsfw: allowsNsfw.value,
          limit: PAGE_SIZE,
          ...(cursor ? { cursor } : {})
        }
      }
    })
  )
  loading.value = false
  if (!result.ok) {
    return
  }
  const seen = new Set(cursor ? items.value.map((item) => item.id) : [])
  items.value = [
    ...(cursor ? items.value : []),
    ...result.data.items.filter((item) => !seen.has(item.id))
  ]
  nextCursor.value = result.data.next_cursor
}

const topicLink = (item: SubscribedTopic) =>
  item.first_unread_floor
    ? `/topic/${item.id}?reply=${item.first_unread_floor}`
    : `/topic/${item.id}`

onMounted(() => {
  void load()
})
watch(allowsNsfw, () => {
  void load()
})
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
      v-if="nextCursor"
      variant="light"
      size="sm"
      :loading="loading"
      @click="load(nextCursor)"
    >
      查看更多
    </KunButton>
  </section>
</template>
