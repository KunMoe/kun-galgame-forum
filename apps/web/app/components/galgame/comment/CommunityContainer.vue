<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const emit = defineEmits<{
  'update:loading': [boolean]
}>()

const route = useRoute()
const api = useApiClient()
const gid = parseInt((route.params as { gid: string }).gid)

const target: CommunityCommentTarget = { kind: 'galgame', galgameId: gid }

const galgame = inject<GalgameDetail>('galgame')

const {
  posts,
  status,
  following,
  setFollowing,
  seeded,
  loadFailed,
  groups,
  isEmpty,
  hasMore,
  loadingMore,
  loadMore,
  handleNewComment,
  handleUpdated,
  handleTombstoned,
  scrollToPost
} = await useCommunityCommentList(target)

watchEffect(() => emit('update:loading', status.value === 'pending'))

const showEmpty = computed(
  () => isEmpty.value && galgame?.is_on_forum !== false
)

const ensureLoadedAndScroll = async (postId: string) => {
  const found = await settle(
    api.GET('/wall-comments/{wall_comment_id}', {
      params: { path: { wall_comment_id: postId } }
    })
  )
  if (
    !found.ok ||
    found.data.subject_type !== 'galgame' ||
    found.data.subject_id !== String(gid)
  ) {
    return
  }
  let guard = 0
  while (
    !posts.value.some((p) => p.id === postId) &&
    hasMore.value &&
    guard < 50
  ) {
    await loadMore()
    guard += 1
  }
  if (posts.value.some((p) => p.id === postId)) {
    scrollToPost(postId)
  }
}

const resolveDeepLink = async () => {
  const commentParam = String(route.query.comment ?? '')
  const hashMatch = (route.hash || '').match(/^#galgame-comment-(\d+)$/)
  const postId = /^\d+$/.test(commentParam) ? commentParam : hashMatch?.[1]
  if (postId) {
    await ensureLoadedAndScroll(postId)
  }
}

onMounted(() => {
  if (seeded.value) {
    resolveDeepLink()
    return
  }
  const stop = watch(seeded, (ready) => {
    if (ready) {
      stop()
      resolveDeepLink()
    }
  })
})
</script>

<template>
  <div class="space-y-5">
    <KunHeader name="游戏评论" scale="h2">
      <template #endContent>
        <div class="flex flex-wrap items-center gap-3">
          <KunLink size="sm" to="/topic/1482">
            Galgame 评论注意事项, 资源失效, 解压密码错误等问题反馈
          </KunLink>
          <CommentCommunitySubscribe
            :following="following"
            :submit="setFollowing"
          />
        </div>
      </template>
    </KunHeader>

    <CommentCommunityComposer :target="target" @submitted="handleNewComment" />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-else-if="loadFailed" description="评论加载失败，请稍后刷新重试" />

    <KunNull
      v-else-if="showEmpty"
      description="没人评论, 是没人要这个 Galgame 的小只可爱软萌女孩子了吗, 呜呜呜呜呜呜！！"
    />

    <div v-else-if="groups.length" class="space-y-8">
      <CommentCommunityRow
        v-for="group in groups"
        :key="group.root.id"
        :comment="group.root"
        :replies="group.replies"
        :target="target"
        :depth="0"
        @reply-added="handleNewComment"
        @updated="handleUpdated"
        @tombstoned="handleTombstoned"
      />
    </div>

    <KunButton
      v-if="hasMore"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多评论
    </KunButton>

    <CommentCommunityFlagModal />
  </div>
</template>
