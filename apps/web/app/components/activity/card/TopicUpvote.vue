<script setup lang="ts">
import { randomUpvoteDescription } from '~/constants/upvote'

const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as TopicActivityData | undefined
)
const topicId = computed(() => data.value?.topic_id ?? 0)
const covers = computed(() => (data.value?.cover_images ?? []).slice(0, 3))

const seed = computed(() => {
  const n = Number(props.activity.unique_id.split(':').pop())
  return Number.isFinite(n) ? n : topicId.value
})
const blurb = computed(
  () => props.activity.content || randomUpvoteDescription(seed.value)
)

const { isFavorited, reactionKeysOf, ensureLoaded } = useMyTopicInteractions()
onMounted(ensureLoaded)

const reactionList = computed<KunReaction[]>(() =>
  (data.value?.reactions ?? []).map((r) => ({
    reaction: r.reaction,
    count: r.count,
    reactors: r.reactors,
    mine: reactionKeysOf(topicId.value).includes(r.reaction)
  }))
)
provide(
  reactionsKey,
  useReactions({
    topicId: topicId.value,
    targetUserId: data.value?.author_id ?? 0,
    reactions: reactionList.value,
    sync: () => reactionList.value,
    showReactors: true
  })
)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-3">
      <p class="text-default-600 text-sm break-all">
        推了这个话题，<span class="text-secondary font-bold">{{ blurb }}</span>
      </p>

      <KunLink
        underline="none"
        color="default"
        :to="activity.link"
        class-name="group block space-y-2.5"
      >
        <h3
          class="group-hover:text-primary line-clamp-2 text-lg font-medium break-all transition-colors"
        >
          {{ data?.title }}
        </h3>
        <p
          v-if="data?.excerpt"
          class="text-default-500 line-clamp-3 text-sm break-all"
        >
          {{ markdownToText(data.excerpt) }}
        </p>
        <TopicCoverGrid
          v-if="covers.length"
          :images="covers"
          :meta="data?.cover_image_meta"
          :nsfw="data?.is_nsfw"
        />
      </KunLink>

      <div class="space-y-2">
        <TopicReactionBar />

        <div class="flex items-center justify-between gap-2">
          <div class="flex min-w-0 items-center gap-1">
            <TopicFooterFavorite
              :topic-id="topicId"
              :favorite-count="data?.favorite_count ?? 0"
              :is-favorite="isFavorited(topicId)"
            />
            <TopicReactionTrigger />
          </div>

          <div
            class="text-default-500 flex shrink-0 items-center gap-3 text-sm"
          >
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:eye" class="size-4" />
              {{ formatNumber(data?.view ?? 0) }}
            </span>
            <KunLink
              underline="none"
              color="default"
              :to="activity.link"
              class-name="text-default-500 hover:text-primary flex items-center gap-0.5 text-sm"
            >
              查看详情
              <KunIcon name="lucide:chevron-right" class="size-4" />
            </KunLink>
          </div>
        </div>
      </div>
    </div>
  </ActivityCardShell>
</template>
