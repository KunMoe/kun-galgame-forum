<script setup lang="ts">
import type {
  DiscussionForumPosting,
  WithContext,
  Person,
  Comment,
  InteractionCounter
} from 'schema-dts'
import type { Topic } from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { firstImageUrl } from '#shared/utils/content/plainText'
import { toKunUser } from '~/utils/userRef'
import { problemMessage } from '#shared/utils/api/message'
import { provideTopicPage } from '~/composables/topic/useTopicPage'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const { sectionLabel } = useTopicSections()

const { allowsNsfw } = useContentStance()

const { isReplyRewriting } = storeToRefs(useTempReplyStore())
const { isEdit } = storeToRefs(useTempReplyStore())

const topicId = computed(() => (route.params as { id: string }).id)
provide('topicId', Number(topicId.value))

const { data, problem, pending, refresh } = useKunApi<Topic>(
  () => `topic:${topicId.value}`,
  (api, { signal }) =>
    api.GET('/topics/{topic_id}', {
      params: { path: { topic_id: topicId.value } },
      signal
    })
)

const { polls, lotteries, upvotes } = provideTopicPage(topicId.value)
const firstScreenPending = computed(
  () =>
    polls.pending.value || lotteries.pending.value || upvotes.pending.value
)

const replaced = ref<Topic>()
watch(data, () => {
  replaced.value = undefined
})
const topic = computed(() => replaced.value ?? data.value)
// Decided once at setup, the gate stayed open when the account's stance
// narrowed after hydration (2026-09-24), so it follows topic and stance.
const revealed = ref(false)
const isShowTopic = computed(
  () => !topic.value?.is_nsfw || allowsNsfw.value || revealed.value
)
provide('refreshTopic', refresh)
provide('replaceTopic', (next: Topic) => {
  replaced.value = next
})

onBeforeRouteLeave(async () => {
  let proceed = true
  if (isReplyRewriting.value) {
    const res =
      await useComponentMessageStore().alert(
        '确认离开界面吗？您的更改将不会保存。'
      )
    if (res) {
      useTempReplyStore().resetRewriteReplyData()
    } else {
      proceed = false
    }
  }
  isEdit.value = false
  return proceed
})

onBeforeMount(() => {
  isEdit.value = false
})

const describe = (topic: Topic) =>
  topic.ai_summary || truncateRunes(contentPlainText(topic.content).trim(), 233)

const jsonLdOf = (topic: Topic): WithContext<DiscussionForumPosting> => {
  const topicUrl = `${kungal.domain.main}/topic/${topic.id}`
  const author = toKunUser(topic.author)

  const authorSchema: Person = {
    '@type': 'Person',
    name: author.name,
    url: `${kungal.domain.main}/user/${topic.author.id}`,
    image: author.avatar
  }

  const interactionStatistics: InteractionCounter[] = [
    {
      '@type': 'InteractionCounter',
      interactionType: {
        '@type': 'CommentAction'
      },
      userInteractionCount: topic.reply_count
    },
    {
      '@type': 'InteractionCounter',
      interactionType: {
        '@type': 'LikeAction'
      },
      userInteractionCount: topic.like_count
    },
    {
      '@type': 'InteractionCounter',
      interactionType: {
        '@type': 'VoteAction'
      },
      userInteractionCount: topic.upvote_count
    }
  ]

  const ba = topic.best_answer
  const baAuthor = ba ? toKunUser(ba.author) : undefined
  const acceptedAnswerSchema: Comment | undefined = ba
    ? {
        '@type': 'Comment',
        text: truncateRunes(contentPlainText(ba.content).trim(), 5000),
        datePublished: new Date(ba.created_at).toISOString(),
        url: `${topicUrl}#k${ba.floor}`,
        author: {
          '@type': 'Person',
          name: baAuthor!.name,
          url: `${kungal.domain.main}/user/${ba.author.id}`,
          image: baAuthor!.avatar
        }
      }
    : undefined

  return {
    '@context': 'https://schema.org',
    '@type': 'DiscussionForumPosting',
    mainEntityOfPage: topicUrl,
    headline: topic.title,
    description: describe(topic),
    image:
      topic.cover_images[0]?.url ||
      firstImageUrl(topic.content) ||
      `${kungal.domain.main}/kungalgame.webp`,
    author: authorSchema,
    datePublished: new Date(topic.created_at).toISOString(),
    dateModified: topic.edited_at
      ? new Date(topic.edited_at).toISOString()
      : new Date(topic.created_at).toISOString(),
    interactionStatistic: interactionStatistics,
    commentCount: topic.reply_count,
    ...(acceptedAnswerSchema && { acceptedAnswer: acceptedAnswerSchema }),
    keywords: sectionLabel(topic.section)
  }
}

useHead(() => ({
  script: data.value
    ? [
        {
          id: 'schema-org-qa-page',
          type: 'application/ld+json',
          innerHTML: jsonLdOf(data.value)
        }
      ]
    : []
}))

useKunSeoMeta(() => {
  const topic = data.value
  if (!topic) {
    return problem.value ? { noindex: true, title: '未找到此话题' } : {}
  }
  if (topic.is_nsfw) {
    // Being signed in used to be enough on its own. The account's stance
    // decides now; 模糊 lets the topic through and masks its covers instead.
    return { noindex: true, title: allowsNsfw.value ? topic.title : '' }
  }
  return {
    title: topic.title,
    description: describe(topic),
    ogCard: { kind: 'topic', id: Number(topic.id) },
    ogType: 'article',
    articleAuthor: [`${kungal.domain.main}/user/${topic.author.id}`],
    articlePublishedTime: new Date(topic.created_at).toString(),
    articleModifiedTime: topic.edited_at
      ? new Date(topic.edited_at).toString()
      : ''
  }
})
</script>

<template>
  <div>
    <template v-if="topic && !firstScreenPending">
      <TopicDetail v-if="isShowTopic" :topic="topic" />

      <KunNsfwGate v-else noun="话题" @reveal="revealed = true" />
    </template>

    <TopicDetailSkeleton v-else-if="pending || (topic && firstScreenPending)" />

    <template v-else-if="problem && problem.status !== 404">
      <KunNull :description="problemMessage(problem)" />
      <div class="flex justify-center">
        <KunButton variant="flat" size="sm" @click="() => refresh()">
          重试
        </KunButton>
      </div>
    </template>

    <KunNull v-else description="未找到这个话题" />
  </div>
</template>
