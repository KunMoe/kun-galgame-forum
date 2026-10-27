<script setup lang="ts">
const route = useRoute()
const newsItemId = computed(() => (route.params as { id: string }).id)

if (!/^\d{1,20}$/.test(newsItemId.value)) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该情报',
    fatal: true
  })
}

const {
  data: item,
  problem,
  pending
} = await useKunApi(
  () => `news-item:${newsItemId.value}`,
  (api, { signal }) =>
    api.GET('/news-items/{news_item_id}', {
      params: { path: { news_item_id: newsItemId.value } },
      signal
    })
)

useKunNotFound(() => problem.value !== null, '未找到该情报')

const { byKey } = useNewsSources()

useKunSeoMeta(() =>
  item.value
    ? {
        title: item.value.title,
        description: item.value.preview,
        ogType: 'article',
        ogImage: item.value.banner?.url,
        articlePublishedTime: item.value.published_at
      }
    : {}
)
</script>

<template>
  <div class="pb-12">
    <NewsDetail v-if="item" :item="item" :source="byKey[item.news_source]" />
    <NewsDetailSkeleton v-else-if="pending" />
  </div>
</template>
