<script setup lang="ts">
import { KUN_TOPIC_SECTION } from '~/constants/topic'
import type {
  Activity,
  ReplyExcerpt,
  TopicSummary
} from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{ activity: Activity; topic: TopicSummary }>()

const digest = computed(() => props.activity.topic_digest)
const topicId = computed(() => Number(props.topic.id))
const topicLink = computed(() => `/topic/${props.topic.id}`)
const covers = computed(() => props.topic.cover_images.slice(0, 3))
const hasBadge = computed(
  () =>
    props.topic.has_best_answer ||
    !!props.topic.mini_apps.length ||
    props.topic.is_nsfw ||
    !!props.topic.upvoted_at
)

const topReply = computed(() => digest.value?.top_reply ?? null)
const bestAnswer = computed(() => digest.value?.best_answer_excerpt ?? null)
const upvotes = computed(() => {
  const latest = digest.value?.latest_upvote
  return latest?.upvoted_at ? [latest] : []
})
const showTopReply = computed(
  () =>
    !!topReply.value && topReply.value.reply_id !== bestAnswer.value?.reply_id
)

const latestReply = computed(() => {
  const r = digest.value?.latest_reply
  if (
    !r ||
    r.reply_id === bestAnswer.value?.reply_id ||
    r.reply_id === topReply.value?.reply_id
  ) {
    return null
  }
  return r
})
const latestComment = computed(() => digest.value?.latest_comment ?? null)
const latest = computed(() => {
  if (latestReply.value) {
    return {
      label: '最新回复',
      to: replyPermalink(topicLink.value, latestReply.value.floor),
      user: toKunUser(latestReply.value.author),
      excerpt: latestReply.value.excerpt_markdown,
      created: latestReply.value.created_at
    }
  }
  if (latestComment.value) {
    return {
      label: '最新评论',
      to: commentPermalink(
        topicLink.value,
        Number(latestComment.value.comment_id)
      ),
      user: toKunUser(latestComment.value.author),
      excerpt: latestComment.value.excerpt_markdown,
      created: latestComment.value.created_at
    }
  }
  return null
})

const replyUser = (r: ReplyExcerpt) => toKunUser(r.author)

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
    <template v-if="digest?.edited_at" #meta>
      <span class="text-default-400 ml-2 flex items-center gap-1 text-xs">
        <KunIcon name="lucide:pencil" class="size-3" />
        {{ formatTimeDifference(digest.edited_at) }}
      </span>
    </template>

    <div class="space-y-3">
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
        />
      </KunLink>

      <div
        v-if="hasBadge || topic.sections.length"
        class="flex flex-wrap items-center gap-1.5"
      >
        <TopicBadgeGroup
          v-if="hasBadge"
          :section="[]"
          :upvote-time="topic.upvoted_at"
          :has-best-answer="topic.has_best_answer"
          :mini-apps="topic.mini_apps"
          :is-n-s-f-w-topic="topic.is_nsfw"
        />
        <KunChip
          v-for="sec in topic.sections"
          :key="sec"
          size="sm"
          variant="flat"
          color="primary"
        >
          {{ KUN_TOPIC_SECTION[sec] }}
        </KunChip>
      </div>

      <TopicUpvoteRecords
        v-if="upvotes.length"
        :topic-id="topic.id"
        :records="upvotes"
      />

      <KunLink
        v-if="latest"
        underline="none"
        color="default"
        :to="latest.to"
        class-name="bg-default-100 flex gap-2 rounded-md p-1.5"
      >
        <div class="bg-default-300 w-1 shrink-0 rounded-full" />
        <div class="min-w-0 flex-1 space-y-1 text-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-1.5">
              <KunAvatar :user="latest.user" size="sm" :is-navigation="false" />
              <span class="text-default-700 line-clamp-1 font-medium">
                {{ latest.user.name }}
              </span>
              <span class="text-default-400 shrink-0 text-xs">
                {{ latest.label }}
              </span>
            </span>
            <span class="text-default-400 shrink-0 text-xs whitespace-nowrap">
              {{ formatTimeDifference(latest.created) }}
            </span>
          </div>
          <p class="text-default-600 line-clamp-2 break-all">
            {{ markdownToText(latest.excerpt) }}
          </p>
        </div>
      </KunLink>

      <KunLink
        v-if="showTopReply && topReply"
        underline="none"
        color="default"
        :to="replyPermalink(topicLink, topReply.floor)"
        class-name="bg-primary-500/10 flex gap-2 rounded-md p-1.5"
      >
        <div class="bg-primary w-1 shrink-0 rounded-full" />
        <div class="min-w-0 flex-1 space-y-1 text-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-1.5">
              <KunAvatar
                :user="replyUser(topReply)"
                size="sm"
                :is-navigation="false"
              />
              <span class="text-default-700 line-clamp-1 font-medium">
                {{ replyUser(topReply).name }}
              </span>
            </span>
            <span class="text-default-500 flex shrink-0 items-center gap-1">
              <KunIcon name="lucide:thumbs-up" class="size-3.5" />
              {{ topReply.like_count }}
            </span>
          </div>
          <p class="text-default-600 line-clamp-2 break-all">
            {{ markdownToText(topReply.excerpt_markdown) }}
          </p>
        </div>
      </KunLink>

      <KunLink
        v-if="bestAnswer"
        underline="none"
        color="default"
        :to="replyPermalink(topicLink, bestAnswer.floor)"
        class-name="bg-success-500/10 relative flex gap-2 overflow-hidden rounded-md p-1.5"
      >
        <div class="bg-success-500 w-1 shrink-0 rounded-full" />
        <div class="min-w-0 flex-1 space-y-1 text-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-1.5">
              <KunAvatar
                :user="replyUser(bestAnswer)"
                size="sm"
                :is-navigation="false"
              />
              <span
                class="text-success-700 dark:text-success-300 line-clamp-1 font-medium"
              >
                {{ replyUser(bestAnswer).name }}
              </span>
            </span>
            <span
              class="text-success-600 dark:text-success-400 flex shrink-0 items-center gap-1"
            >
              <KunIcon name="lucide:thumbs-up" class="size-3.5" />
              {{ bestAnswer.like_count }}
            </span>
          </div>
          <p class="text-default-600 line-clamp-2 break-all">
            {{ markdownToText(bestAnswer.excerpt_markdown) }}
          </p>
        </div>
        <KunIcon
          name="lucide:circle-check-big"
          class-name="text-success-500/20 pointer-events-none absolute right-1 bottom-0 size-14"
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
              <KunIcon name="lucide:message-square" class="size-4" />
              {{ formatNumber(topic.reply_count + topic.comment_count) }}
            </span>
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
