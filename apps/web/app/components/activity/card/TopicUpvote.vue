<script setup lang="ts">
import { randomUpvoteDescription } from '~/constants/upvote'
import type { Activity, TopicSummary } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{ activity: Activity; topic: TopicSummary }>()

const digest = computed(() => props.activity.topic_digest)
const topicId = computed(() => Number(props.topic.id))
const topicLink = computed(() => `/topic/${props.topic.id}`)
const covers = computed(() => props.topic.cover_images.slice(0, 3))

const blurb = computed(
  () =>
    props.activity.excerpt_markdown || randomUpvoteDescription(topicId.value)
)

const { isFavorited, reactionKeysOf, ensureLoaded } = useMyTopicInteractions()
onMounted(() => ensureLoaded([topicId.value]))

const reactionList = computed<KunReaction[]>(() =>
  (digest.value?.reactions ?? []).map((r) => ({
    reaction: r.reaction,
    count: r.count,
    reactors: r.reactors.map(toKunUser),
    mine: reactionKeysOf(topicId.value).includes(r.reaction)
  }))
)
provide(
  reactionsKey,
  useReactions({
    topicId: topicId.value,
    targetUserId: Number(props.topic.author.id),
    reactions: reactionList.value,
    sync: () => reactionList.value,
    showReactors: true
  })
)
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-3">
      <p class="text-default-600 text-sm break-all">
        推了这个话题，<span class="text-secondary font-bold">{{ blurb }}</span>
      </p>

      <KunLink
        underline="none"
        color="default"
        :to="topicLink"
        class-name="group block space-y-2.5"
      >
        <h3
          class="group-hover:text-primary line-clamp-2 text-lg font-medium break-all transition-colors"
        >
          {{ topic.title }}
        </h3>
        <p
          v-if="digest?.excerpt_markdown"
          class="text-default-500 line-clamp-3 text-sm break-all"
        >
          {{ markdownToText(digest.excerpt_markdown) }}
        </p>
        <TopicCoverGrid
          v-if="covers.length"
          :images="covers"
          :nsfw="topic.is_nsfw"
          compact
        />
      </KunLink>

      <div class="space-y-2">
        <TopicReactionBar />

        <div class="flex items-center justify-between gap-2">
          <div class="flex min-w-0 items-center gap-1">
            <TopicFooterFavorite
              :topic-id="topicId"
              :favorite-count="digest?.favorite_count ?? 0"
              :is-favorite="isFavorited(topicId)"
            />
            <TopicReactionTrigger />
          </div>

          <div
            class="text-default-500 flex shrink-0 items-center gap-3 text-sm"
          >
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:eye" class="size-4" />
              {{ formatNumber(topic.view_count) }}
            </span>
            <KunLink
              underline="none"
              color="default"
              :to="topicLink"
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
