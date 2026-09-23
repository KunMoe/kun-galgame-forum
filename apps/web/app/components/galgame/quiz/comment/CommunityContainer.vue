<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'

const props = defineProps<{
  quizId: number
  commentCount: number
}>()

const {
  status,
  following,
  setFollowing,
  seeded,
  locked,
  loadFailed,
  groups,
  isEmpty,
  total,
  hasMore,
  loadingMore,
  loadMore,
  handleNewComment,
  handleUpdated,
  handleTombstoned,
  scrollToPost
} = await useCommunityCommentList(
  { kind: 'quiz', quizId: props.quizId },
  props.commentCount
)

const target: CommunityCommentTarget = { kind: 'quiz', quizId: props.quizId }

const onPublished = (post: WallComment) => {
  handleNewComment(post)
  if (post.root_comment_id == null) {
    scrollToPost(post.id)
  }
}
</script>

<template>
  <KunCard :is-transparent="false" :is-hoverable="false">
    <KunHeader
      name="题目讨论"
      description="聊聊这道题目, 请不要直接剧透答案"
      scale="h2"
    >
      <template #endContent>
        <div class="flex items-center gap-3">
          <span v-if="total > 0 && !locked" class="text-default-500 text-sm">
            {{ total }} 条讨论
          </span>
          <CommentCommunitySubscribe
            v-if="!locked"
            :following="following"
            :submit="setFollowing"
          />
        </div>
      </template>
    </KunHeader>

    <KunLoading v-if="status === 'pending' && !seeded" />

    <KunNull
      v-else-if="locked"
      description="这道题目含有剧透, 作答后即可查看并参与讨论"
    />

    <div v-else class="space-y-5">
      <CommentCommunityComposer :target="target" @submitted="onPublished" />

      <KunNull
        v-if="loadFailed"
        description="讨论加载失败，请稍后刷新重试"
      />

      <KunNull v-else-if="isEmpty" description="还没有人讨论这道题目" />

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
        加载更多讨论
      </KunButton>
    </div>

    <CommentCommunityFlagModal />
  </KunCard>
</template>
