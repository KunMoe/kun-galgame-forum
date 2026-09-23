<script setup lang="ts">
const props = defineProps<{
  toolsetId: number
}>()

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
  handleTombstoned
} = await useCommunityCommentList({
  kind: 'toolset',
  toolsetId: props.toolsetId
})

const target: CommunityCommentTarget = {
  kind: 'toolset',
  toolsetId: props.toolsetId
}
</script>

<template>
  <div class="space-y-5">
    <KunHeader
      name="评论"
      description="如果您对该工具有任何的使用疑问, 欢迎发布评论"
      scale="h2"
    >
      <template #endContent>
        <CommentCommunitySubscribe
          :following="following"
          :submit="setFollowing"
        />
      </template>
    </KunHeader>

    <CommentCommunityComposer :target="target" @submitted="handleNewComment" />

    <KunLoading v-if="status === 'pending' && !seeded" />

    <KunNull v-else-if="loadFailed" description="评论加载失败，请稍后刷新重试" />

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

    <CommentCommunityFlagModal />
  </div>
</template>
