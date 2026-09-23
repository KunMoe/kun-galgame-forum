<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'

const props = defineProps<{
  resourceId: number
  commentCount: number
}>()

const {
  status,
  following,
  setFollowing,
  seeded,
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
  { kind: 'resource', resourceId: props.resourceId },
  props.commentCount
)

const target: CommunityCommentTarget = {
  kind: 'resource',
  resourceId: props.resourceId
}

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
      name="资源评论"
      description="这个资源能正常使用吗? 有问题欢迎在这里反馈"
      scale="h2"
    >
      <template #endContent>
        <div class="flex items-center gap-3">
          <span v-if="total > 0" class="text-default-500 text-sm">
            {{ total }} 条评论
          </span>
          <CommentCommunitySubscribe
            :following="following"
            :submit="setFollowing"
          />
        </div>
      </template>
    </KunHeader>

    <div class="space-y-5">
      <CommentCommunityComposer :target="target" @submitted="onPublished" />

      <KunLoading v-if="status === 'pending' && !seeded" />

      <KunNull
        v-else-if="loadFailed"
        description="评论加载失败，请稍后刷新重试"
      />

      <KunNull v-else-if="isEmpty" description="还没有人评论这个资源" />

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
    </div>

    <CommentCommunityFlagModal />
  </KunCard>
</template>
