<script setup lang="ts">
import type { CollectionSummary } from '#shared/utils/api/schemas'
import { useCursorList } from '~/composables/useCursorList'

const { allowsNsfw, stanceKey } = useContentStance()

const { items, hasMore, pending, loadingMore, loadMore, problem } =
  useCursorList<CollectionSummary>(
    () => `followed-collections:${stanceKey.value}`,
    (api, cursor, { signal }) =>
      api.GET('/me/followed-collections', {
        params: {
          query: {
            include_nsfw: allowsNsfw.value,
            limit: 24,
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )
</script>

<template>
  <div class="space-y-3">
    <GalgameCollectionRowCardSkeleton
      v-if="pending && !items.length"
      :count="4"
      :columns="1"
    />

    <template v-else-if="items.length">
      <GalgameCollectionRowCard
        show-new-items
        v-for="c in items"
        :key="c.id"
        :collection="c"
      />
      <KunButton
        v-if="hasMore"
        variant="light"
        color="primary"
        full-width
        :loading="loadingMore"
        @click="loadMore"
      >
        <KunIcon name="lucide:chevron-down" />
        加载更多
      </KunButton>
    </template>

    <KunNull v-else-if="problem" description="加载失败，请稍后重试" />
    <KunNull
      v-else
      description="你还没有关注任何收藏夹，在收藏夹页面点击「关注」即可"
    />
  </div>
</template>
