<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'

const props = defineProps<{
  collectionId: string
}>()

const target: CommunityCommentTarget = {
  kind: 'collection',
  collectionId: props.collectionId
}

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
} = useCommunityCommentList(target)

const onPublished = (post: WallComment) => {
  handleNewComment(post)
  if (post.root_comment_id == null) {
    scrollToPost(post.id)
  }
}
</script>

<template>
  <section class="space-y-5">
    <KunHeader
      name="评论区"
      description="这里的评论在所有 NextMoe 站点共享"
      scale="h3"
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

    <CommentCommunityComposer :target="target" @submitted="onPublished" />

    <CommentCommunityListSkeleton v-if="!seeded" />

    <KunNull
      v-else-if="loadFailed"
      description="评论加载失败，请稍后刷新重试"
    />

    <KunNull v-else-if="isEmpty" description="还没有人评论这个收藏夹" />

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
  </section>
</template>
