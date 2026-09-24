<script setup lang="ts">
import { KUN_TOPIC_SECTION } from '~/constants/topic'
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

definePageMeta({ key: (route) => route.path })

const route = useRoute()

const { allowsNsfw } = useContentStance()

const { isReplyRewriting } = storeToRefs(useTempReplyStore())
const { isEdit } = storeToRefs(useTempReplyStore())

const topicId = computed(() => (route.params as { id: string }).id)
provide('topicId', Number(topicId.value))

const { data, problem, refresh } = await useApi<Topic>(
  () => `topic:${topicId.value}`,
  (api, { signal }) =>
    api.GET('/topics/{topic_id}', {
      params: { path: { topic_id: topicId.value } },
      signal
    })
)

const topic = ref<Topic | undefined>(data.value)
watch(data, (next) => {
  if (next) {
    topic.value = next
  }
})
// Decided once at setup, the gate stayed open when the account's stance
// narrowed after hydration (2026-09-24), so it follows topic and stance.
const revealed = ref(false)
const isShowTopic = computed(
  () => !topic.value?.is_nsfw || allowsNsfw.value || revealed.value
)
provide('refreshTopic', refresh)
provide('replaceTopic', (next: Topic) => {
  topic.value = next
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

if (data.value) {
  const topic = data.value
  const author = toKunUser(topic.author)
  const banner =
    topic.cover_images[0]?.url ||
    firstImageUrl(topic.content) ||
    `${kungal.domain.main}/kungalgame.webp`
  const created = new Date(topic.created_at).toString()
  const updated = topic.edited_at ? new Date(topic.edited_at).toString() : ''
  const description = computed(() =>
    truncateRunes(contentPlainText(topic.content).trim(), 233)
  )

  const jsonLd = computed<WithContext<DiscussionForumPosting>>(() => {
    const topicUrl = `${kungal.domain.main}/topic/${topic.id}`

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
      description: description.value,
      image: banner,
      author: authorSchema,
      datePublished: new Date(topic.created_at).toISOString(),
      dateModified: topic.edited_at
        ? new Date(topic.edited_at).toISOString()
        : new Date(topic.created_at).toISOString(),
      interactionStatistic: interactionStatistics,
      commentCount: topic.reply_count,
      ...(acceptedAnswerSchema && { acceptedAnswer: acceptedAnswerSchema }),
      keywords: [
        ...topic.sections.map((s) => KUN_TOPIC_SECTION[s]).filter(Boolean)
      ].join(', ')
    }
  })

  useHead({
    script: [
      {
        id: 'schema-org-qa-page',
        type: 'application/ld+json',
        innerHTML: jsonLd.value
      }
    ]
  })

  if (topic.is_nsfw) {
    // Being signed in used to be enough on its own. The account's stance
    // decides now; 模糊 lets the topic through and masks its covers instead.
    useKunDisableSeo(allowsNsfw.value ? topic.title : '')
  } else {
    useKunSeoMeta({
      title: topic.title,
      description: description.value,
      ogCard: { kind: 'topic', id: Number(topic.id) },
      ogType: 'article',
      articleAuthor: [`${kungal.domain.main}/user/${topic.author.id}`],
      articlePublishedTime: created,
      articleModifiedTime: updated
    })
  }
} else {
  useKunDisableSeo('未找到此话题')
}
</script>

<template>
  <div>
    <template v-if="topic">
      <TopicDetail v-if="isShowTopic" :topic="topic" />

      <KunNsfwGate v-else noun="话题" @reveal="revealed = true" />
    </template>

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
