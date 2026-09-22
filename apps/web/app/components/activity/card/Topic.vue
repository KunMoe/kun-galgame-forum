<script setup lang="ts">
import { KUN_TOPIC_SECTION } from '~/constants/topic'

const props = defineProps<{ activity: ActivityItem }>()

const data = computed(
  () => props.activity.data as TopicActivityData | undefined
)
const covers = computed(() => (data.value?.cover_images ?? []).slice(0, 3))
const topicId = computed(() => data.value?.topic_id ?? 0)
const hasBadge = computed(() => {
  const d = data.value
  return (
    !!d &&
    (d.has_best_answer || !!d.mini_apps?.length || d.is_nsfw || !!d.upvote_time)
  )
})

const topReply = computed(() => data.value?.top_reply)
const bestAnswer = computed(() => data.value?.best_answer)
const upvotes = computed(() => {
  const all = data.value?.upvotes ?? []
  if (all.length <= 1) return all
  return [...all]
    .sort(
      (a, b) => new Date(b.created).getTime() - new Date(a.created).getTime()
    )
    .slice(0, 1)
})
const sameReply = computed(
  () =>
    !!bestAnswer.value &&
    !!topReply.value &&
    bestAnswer.value.reply_id === topReply.value.reply_id
)
const showTopReply = computed(() => !!topReply.value && !sameReply.value)

const latest = computed(() => data.value?.latest_activity)
const showLatest = computed(() => {
  const l = latest.value
  if (!l) return false
  if (
    l.kind === 'reply' &&
    (l.reply_id === bestAnswer.value?.reply_id ||
      l.reply_id === topReply.value?.reply_id)
  ) {
    return false
  }
  return true
})

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
    targetUserId: props.activity.actor?.id ?? 0,
    reactions: reactionList.value,
    sync: () => reactionList.value,
    showReactors: true
  })
)
</script>

<template>
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <template v-if="data?.edited" #meta>
      <span class="text-default-400 ml-2 flex items-center gap-1 text-xs">
        <KunIcon name="lucide:pencil" class="size-3" />
        {{ formatTimeDifference(data.edited) }}
      </span>
    </template>

    <div class="space-y-3">
      <KunLink
        underline="none"
        color="default"
        :to="activity.link"
        class-name="group block space-y-2.5"
      >
        <h3
          class="group-hover:text-primary line-clamp-2 text-lg font-medium break-all transition-colors"
        >
          {{ activity.content }}
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

      <div
        v-if="hasBadge || data?.sections?.length"
        class="flex flex-wrap items-center gap-1.5"
      >
        <TopicBadgeGroup
          v-if="hasBadge"
          :section="[]"
          :upvote-time="data?.upvote_time"
          :has-best-answer="data?.has_best_answer"
          :mini-apps="data?.mini_apps"
          :is-n-s-f-w-topic="data?.is_nsfw"
        />
        <KunChip
          v-for="(sec, index) in data?.sections ?? []"
          :key="index"
          size="sm"
          variant="flat"
          color="primary"
        >
          {{ KUN_TOPIC_SECTION[sec] }}
        </KunChip>
      </div>

      <TopicUpvoteRecords v-if="upvotes.length" :records="upvotes" />

      <KunLink
        v-if="showLatest && latest"
        underline="none"
        color="default"
        :to="
          latest?.kind === 'reply'
            ? replyPermalink(activity.link, latest.floor)
            : commentPermalink(activity.link, latest?.comment_id)
        "
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
                {{ latest.kind === 'reply' ? '最新回复' : '最新评论' }}
              </span>
            </span>
            <span class="text-default-400 shrink-0 text-xs whitespace-nowrap">
              {{ formatTimeDifference(latest.created) }}
            </span>
          </div>
          <p class="text-default-600 line-clamp-2 break-all">
            {{ markdownToText(latest.content) }}
          </p>
        </div>
      </KunLink>

      <KunLink
        v-if="showTopReply && topReply"
        underline="none"
        color="default"
        :to="replyPermalink(activity.link, topReply?.floor)"
        class-name="bg-primary-500/10 flex gap-2 rounded-md p-1.5"
      >
        <div class="bg-primary w-1 shrink-0 rounded-full" />
        <div class="min-w-0 flex-1 space-y-1 text-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-1.5">
              <KunAvatar
                :user="topReply.user"
                size="sm"
                :is-navigation="false"
              />
              <span class="text-default-700 line-clamp-1 font-medium">
                {{ topReply.user.name }}
              </span>
            </span>
            <span class="text-default-500 flex shrink-0 items-center gap-1">
              <KunIcon name="lucide:thumbs-up" class="size-3.5" />
              {{ topReply.like_count }}
            </span>
          </div>
          <p class="text-default-600 line-clamp-2 break-all">
            {{ markdownToText(topReply.content) }}
          </p>
        </div>
      </KunLink>

      <KunLink
        v-if="bestAnswer"
        underline="none"
        color="default"
        :to="replyPermalink(activity.link, bestAnswer?.floor)"
        class-name="bg-success-500/10 relative flex gap-2 overflow-hidden rounded-md p-1.5"
      >
        <div class="bg-success-500 w-1 shrink-0 rounded-full" />
        <div class="min-w-0 flex-1 space-y-1 text-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="flex min-w-0 items-center gap-1.5">
              <KunAvatar
                :user="bestAnswer.user"
                size="sm"
                :is-navigation="false"
              />
              <span
                class="text-success-700 dark:text-success-300 line-clamp-1 font-medium"
              >
                {{ bestAnswer.user.name }}
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
            {{ markdownToText(bestAnswer.content) }}
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
              :favorite-count="data?.favorite_count ?? 0"
              :is-favorite="isFavorited(topicId)"
            />
            <TopicReactionTrigger />
          </div>

          <div
            class="text-default-500 flex shrink-0 items-center gap-3 text-sm"
          >
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:message-square" class="size-4" />
              {{
                formatNumber(
                  (data?.reply_count ?? 0) + (data?.comment_count ?? 0)
                )
              }}
            </span>
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
