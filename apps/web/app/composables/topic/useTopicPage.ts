import type { InjectionKey } from 'vue'
import { settle } from '#shared/utils/api/problem'
import type { TopicUpvote } from '#shared/utils/api/schemas'
import { replyPageStart, useTopicReplies } from './useTopicReplies'

const startInitialReplies = (topicId: string) => {
  const route = useRoute()
  const api = useApiClient()
  const replyFloor = Number(route.query.reply) || 0
  const commentId = Number(route.query.comment) || 0
  const replyTarget = useState(
    `kun-topic-reply-target-${topicId}:${replyFloor}:${commentId}`,
    () => replyFloor
  )
  const { loadInitialReplies } = useTopicReplies(topicId, api)

  const ready = (async () => {
    if (replyTarget.value <= 0 && commentId > 0) {
      const located = await settle(
        api.GET('/comments/{comment_id}', {
          params: { path: { comment_id: String(commentId) } }
        })
      )
      if (located.ok) {
        replyTarget.value = located.data.reply_floor
      }
    }
    await loadInitialReplies({
      fromFloor:
        replyTarget.value > 0 ? replyPageStart(replyTarget.value) : undefined
    })
  })()
  onServerPrefetch(() => ready)

  return replyTarget
}

export const useTopicUpvotes = (topicId: string) =>
  useCursorList<TopicUpvote>(
    () => `topic-upvotes:${topicId}`,
    (api, cursor) =>
      api.GET('/topics/{topic_id}/upvotes', {
        params: {
          path: { topic_id: topicId },
          query: { limit: 20, ...(cursor ? { cursor } : {}) }
        }
      })
  )

const createTopicPageRequests = (topicId: string) => {
  const { allowsNsfw, stanceKey } = useContentStance()
  return {
    polls: useKunApi(
      () => `topic-polls:${topicId}`,
      (api, { signal }) =>
        api.GET('/topics/{topic_id}/polls', {
          params: { path: { topic_id: topicId } },
          signal
        })
    ),
    lotteries: useKunApi(
      () => `topic-lotteries:${topicId}:${stanceKey.value}`,
      (api, { signal }) =>
        api.GET('/topics/{topic_id}/lotteries', {
          params: {
            path: { topic_id: topicId },
            query: { include_nsfw: allowsNsfw.value }
          },
          signal
        })
    ),
    upvotes: useTopicUpvotes(topicId),
    replyTarget: startInitialReplies(topicId)
  }
}

type TopicPageRequests = ReturnType<typeof createTopicPageRequests>
const TOPIC_PAGE: InjectionKey<TopicPageRequests> = Symbol('topic-page')

// The page starts everything between the topic body and the replies in
// parallel with the topic itself; a section that fetched on mount only
// started once the topic had landed, and pushed the replies down when it did.
export const provideTopicPage = (topicId: string) => {
  const requests = createTopicPageRequests(topicId)
  provide(TOPIC_PAGE, requests)
  return requests
}

export const useOptionalTopicPage = () => inject(TOPIC_PAGE, null)

export const useTopicPage = () => {
  const requests = inject(TOPIC_PAGE)
  if (!requests) {
    throw new Error('useTopicPage outside a topic page')
  }
  return requests
}
