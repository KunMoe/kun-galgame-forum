<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import type { PageListUserTopicItem } from '#shared/utils/api/schemas'

const props = defineProps<{
  userId: number
  userName: string
  topicId: string
}>()

const SHOWN = 5

const { allowsNsfw } = useContentStance()
const isWide = useMediaQuery('(min-width: 1280px)')

const { data, status, refresh } = useKunApi<PageListUserTopicItem>(
  () =>
    `user-topics:${props.userId}:authored:1:${SHOWN + 1}:${allowsNsfw.value ? 'nsfw' : 'sfw'}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/topics', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: 'authored',
          page: 1,
          limit: SHOWN + 1,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    }),
  { server: false, lazy: true, immediate: false }
)

watch(
  isWide,
  (wide) => {
    if (wide && status.value === 'idle') {
      void refresh()
    }
  },
  { immediate: true }
)

const topics = computed(() =>
  (data.value?.items ?? [])
    .filter((t) => t.id !== props.topicId)
    .slice(0, SHOWN)
)
</script>

<template>
  <section
    v-if="topics.length"
    class="flex w-full min-w-0 flex-col items-stretch gap-2"
  >
    <h3 class="text-sm font-medium break-all">{{ userName }}的其它话题</h3>
    <ul class="flex flex-col gap-1">
      <li v-for="t in topics" :key="t.id">
        <KunLink
          :to="`/topic/${t.id}`"
          underline="none"
          color="default"
          class-name="hover:bg-default-100 flex w-full flex-col items-start gap-0.5 rounded-lg px-2 py-1.5 text-left transition-colors"
        >
          <span class="line-clamp-2 text-sm leading-snug">{{ t.title }}</span>
          <KunTime
            :time="t.created_at"
            type="relative"
            class="text-default-500 text-xs"
          />
        </KunLink>
      </li>
    </ul>
    <KunLink
      :to="`/user/${userId}/topic/topic`"
      underline="hover"
      color="default"
      class-name="text-default-500 px-2 text-xs"
    >
      查看全部
    </KunLink>
  </section>
</template>
