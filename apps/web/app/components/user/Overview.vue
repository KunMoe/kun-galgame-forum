<script setup lang="ts">
import type {
  PageListUserCommentItem,
  PageListUserReplyItem,
  PageListUserTopicItem
} from '#shared/utils/api/schemas'

const props = defineProps<{
  userId: number
}>()

const settings = usePersistSettingsStore()
const { allowsNsfw: includeNsfw } = useContentStance()
const uid = props.userId
const ownerId = String(uid)
const N = 4

const topics = useApi<PageListUserTopicItem>(
  () =>
    `user-topics:${ownerId}:authored:1:${N}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/topics', {
      params: {
        path: { user_id: ownerId },
        query: {
          relation: 'authored',
          page: 1,
          limit: N,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    })
)
const galgames = useKunFetch<{ items: GalgameCard[]; total: number }>(
  `/user/${uid}/galgames`,
  {
    query: {
      page: 1,
      limit: N,
      type: 'galgame_publish',
      user_id: uid,
      show_no_resource: settings.showKUNGalgameNoResource
    }
  }
)
const ratings = useKunFetch<{ rating_data: GalgameRatingCard[]; total: number }>(
  `/user/${uid}/ratings`,
  { query: { page: 1, limit: N, user_id: uid } }
)
const resources = useKunFetch<{
  resources: UserGalgameResource[]
  total: number
}>(`/user/${uid}/resources`, {
  query: { page: 1, limit: N, type: 'valid', user_id: uid }
})
const replies = useApi<PageListUserReplyItem>(
  () =>
    `user-replies:${ownerId}:authored:1:${N}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/replies', {
      params: {
        path: { user_id: ownerId },
        query: {
          relation: 'authored',
          page: 1,
          limit: N,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    })
)
const comments = useApi<PageListUserCommentItem>(
  () =>
    `user-comments:${ownerId}:authored:1:${N}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (api, { signal }) =>
    api.GET('/users/{user_id}/comments', {
      params: {
        path: { user_id: ownerId },
        query: {
          relation: 'authored',
          page: 1,
          limit: N,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    })
)
await Promise.all([topics, galgames, ratings, resources, replies, comments])

const topicItems = computed(() =>
  (topics.data.value?.items ?? []).map((t) => ({
    text: t.title,
    time: t.created_at,
    href: `/topic/${t.id}`
  }))
)
const resourceItems = computed(() =>
  (resources.data.value?.resources ?? []).map((r) => ({
    text: r.galgame_name,
    time: r.created,
    href: `/galgame/${r.galgame_id}`
  }))
)
const replyItems = computed(() =>
  (replies.data.value?.items ?? []).map((r) => ({
    text: r.excerpt,
    time: r.created_at,
    href: replyPermalink(`/topic/${r.topic_id}`, r.floor)
  }))
)
const commentItems = computed(() =>
  (comments.data.value?.items ?? []).map((c) => ({
    text: c.excerpt,
    time: c.created_at,
    href: commentPermalink(`/topic/${c.topic_id}`, c.id)
  }))
)
const galgameItems = computed(() => galgames.data.value?.items ?? [])
const ratingItems = computed(() => ratings.data.value?.rating_data ?? [])

const isEmpty = computed(
  () =>
    !topicItems.value.length &&
    !galgameItems.value.length &&
    !ratingItems.value.length &&
    !resourceItems.value.length &&
    !replyItems.value.length &&
    !commentItems.value.length
)
</script>

<template>
  <div class="space-y-8">
    <KunNull v-if="isEmpty" description="这只笨蛋萝莉还没有任何动态" />

    <UserOverviewSection
      v-if="topicItems.length"
      title="话题"
      icon="lucide:square-gantt-chart"
      :view-all-href="`/user/${uid}/topic/topic`"
      :items="topicItems"
    />

    <UserOverviewSection
      v-if="galgameItems.length"
      title="Galgame"
      icon="lucide:gamepad-2"
      :view-all-href="`/user/${uid}/galgame/galgame-publish`"
    >
      <GalgameCard :is-transparent="false" :galgames="galgameItems" />
    </UserOverviewSection>

    <UserOverviewSection
      v-if="ratingItems.length"
      title="Gal 评分"
      icon="lucide:star"
      :view-all-href="`/user/${uid}/rating`"
    >
      <GalgameRatingCard :ratings="ratingItems" :is-transparent="false" />
    </UserOverviewSection>

    <UserOverviewSection
      v-if="resourceItems.length"
      title="Gal 资源"
      icon="lucide:package"
      :view-all-href="`/user/${uid}/resource/valid`"
      :items="resourceItems"
    />

    <UserOverviewSection
      v-if="replyItems.length"
      title="回复"
      icon="carbon:reply"
      :view-all-href="`/user/${uid}/reply/reply-created`"
      :items="replyItems"
    />

    <UserOverviewSection
      v-if="commentItems.length"
      title="评论"
      icon="uil:comment-dots"
      :view-all-href="`/user/${uid}/comment/comment-created`"
      :items="commentItems"
    />
  </div>
</template>
