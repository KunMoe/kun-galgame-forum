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

const { data: item } = await useApi(
  () => `news-item:${newsItemId.value}`,
  (api, { signal }) =>
    api.GET('/news-items/{news_item_id}', {
      params: { path: { news_item_id: newsItemId.value } },
      signal
    })
)

if (!item.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该情报',
    fatal: true
  })
}

const { byKey } = await useNewsSources()

useKunSeoMeta({
  title: item.value.title,
  description: item.value.preview,
  ogType: 'article',
  articlePublishedTime: item.value.published_at
})
</script>

<template>
  <div class="pb-12">
    <NewsDetail v-if="item" :item="item" :source="byKey[item.news_source]" />
  </div>
</template>
