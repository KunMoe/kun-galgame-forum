<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'

const props = defineProps<{
  websiteId: number
}>()

const target: CommunityCommentTarget = {
  kind: 'website',
  websiteId: props.websiteId
}

const {
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
      name="用户评论"
      description="说说你对这个网站的使用体验"
      scale="h2"
    >
      <template #endContent>
        <CommentCommunitySubscribe
          :following="following"
          :submit="setFollowing"
        />
      </template>
    </KunHeader>

    <div class="space-y-5">
      <CommentCommunityComposer :target="target" @submitted="onPublished" />

      <KunLoading v-if="status === 'pending' && !seeded" />

      <KunNull
        v-else-if="loadFailed"
        description="评论加载失败，请稍后刷新重试"
      />

      <KunNull v-else-if="isEmpty" />

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
