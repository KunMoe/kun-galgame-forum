<script setup lang="ts">
import ContentDocument from '~/components/content/Document.vue'
import type {
  ReplyEngagement,
  Topic,
  TopicEngagement
} from '#shared/utils/api/schemas'
import { documentImageHashes } from '#shared/utils/content/plainText'
import { toKunReactions } from '~/utils/reactionSummary'
import { toKunUserWithPoints } from '~/utils/userRef'

const props = defineProps<{
  topic: Topic
}>()

const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})

const unseenCovers = computed(() => {
  const hashes = documentImageHashes(props.topic.content)
  return props.topic.cover_images.filter((image) => !hashes.has(image.hash))
})

const applyEngagement = (engagement: TopicEngagement | ReplyEngagement) => {
  if (engagement.object !== 'topic_engagement') {
    return
  }
  replaceTopic(mergeTopicEngagement(props.topic, engagement))
}

provide(
  reactionsKey,
  useReactions({
    topicId: props.topic.id,
    targetUserId: Number(props.topic.author.id),
    reactions: toKunReactions(props.topic.reactions),
    sync: () => toKunReactions(props.topic.reactions),
    showReactors: true,
    onEngagement: applyEngagement
  })
)
</script>

<template>
  <div id="0" class="outline-primary rounded-lg outline-offset-2">
    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      class-name="w-full min-w-0"
      content-class="gap-4 justify-start"
    >
      <header class="space-y-3">
        <h1
          class="text-3xl leading-tight font-bold tracking-tight break-words lg:text-4xl"
        >
          {{ topic.title }}
        </h1>

        <TopicBadgeGroup
          :section="topic.sections"
          :upvote-time="topic.upvoted_at"
          :has-best-answer="false"
          :mini-apps="topic.mini_apps"
          :is-n-s-f-w-topic="topic.is_nsfw"
          :is-nav-to-section="true"
        />

        <div
          class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
        >
          <span class="flex items-center gap-1.5">
            <KunIcon name="lucide:eye" class="size-4" />
            {{ topic.view_count }}
          </span>
          <span class="flex items-center gap-1.5">
            <KunIcon name="lucide:clock" class="size-4" />
            <KunTime :time="topic.created_at" type="datetime" show-year />
          </span>
          <span v-if="topic.edited_at" class="flex items-center gap-1.5">
            <KunIcon name="lucide:pencil-line" class="size-4" />
            编辑于
            <KunTime :time="topic.edited_at" type="datetime" show-year />
          </span>
        </div>
      </header>

      <TopicDetailBestAnswer
        v-if="topic.best_answer"
        :best-answer="topic.best_answer"
      />

      <TopicDetailUser
        class-name="lg:hidden"
        :user="toKunUserWithPoints(topic.author, topic.author_moemoepoint)"
        :created="topic.created_at"
        :edited="topic.edited_at"
        :topic-id="Number(topic.id)"
        :floor="0"
        :show-addition="false"
      />

      <KunDivider />

      <TopicCoverGrid
        v-if="unseenCovers.length"
        :images="unseenCovers"
        :nsfw="topic.is_nsfw"
        zoomable
      />

      <ContentDocument class-name="kun-master" :document="topic.content" />

      <KunDivider />

      <TopicUpvoteRecords :topic-id="topic.id" />

      <div class="flex flex-wrap items-center gap-1.5">
        <TopicReactionBar />
        <span class="md:hidden">
          <TopicReactionTrigger />
        </span>
      </div>

      <p class="text-default-500 ml-auto text-sm">
        本文版权遵循
        <KunLink
          underline="hover"
          size="sm"
          class-name="text-default-500"
          target="_blank"
          rel="noopener noreferrer"
          to="https://creativecommons.org/licenses/by-nc/4.0/deed.en"
        >
          CC BY-NC 协议
        </KunLink>
        和
        <KunLink
          underline="hover"
          size="sm"
          class-name="text-default-500"
          to="/doc/article-copyright"
        >
          本站版权政策
        </KunLink>
      </p>

      <TopicFooter :topic="topic" />
    </KunCard>
  </div>
</template>
