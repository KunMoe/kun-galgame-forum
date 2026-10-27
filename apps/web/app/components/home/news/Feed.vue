<script setup lang="ts">
const {
  items,
  groups,
  status,
  error,
  hasMore,
  isLoadingMore,
  loadMore,
  sentinel
} = await useNewsFeed()
</script>

<template>
  <KunLoadingDim
    class="min-w-0"
    :loading="status === 'pending' && items.length > 0"
  >
    <NewsCardSkeletonList
      v-if="status === 'pending' && !items.length"
      :count="5"
    />
    <KunNull v-else-if="error" description="情报服务暂时不可用，请稍后再试" />
    <KunNull
      v-else-if="status !== 'pending' && !items.length"
      description="暂无情报"
    />

    <div v-else class="space-y-8">
      <div class="flex justify-end">
        <KunLink
          to="/news"
          color="primary"
          size="sm"
          underline="hover"
          is-show-anchor-icon
        >
          查看全部 Galgame 情报
        </KunLink>
      </div>

      <section v-for="group in groups" :key="group.key" class="space-y-3">
        <NewsGroupHeader :group="group" />

        <div class="space-y-3">
          <NewsCard
            v-for="item in group.items"
            :key="item.id"
            :item="item"
            :source="group.source"
          />
        </div>
      </section>

      <NewsCardSkeletonList v-if="isLoadingMore" />
    </div>

    <div v-if="items.length" ref="sentinel" class="flex justify-center pt-6">
      <KunButton
        v-if="hasMore && !isLoadingMore"
        variant="light"
        @click="loadMore(false)"
      >
        加载更多
      </KunButton>
      <span v-else-if="!hasMore" class="text-default-400 text-sm">
        没有更多情报了
      </span>
    </div>
  </KunLoadingDim>
</template>
