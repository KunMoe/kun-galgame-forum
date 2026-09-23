<script setup lang="ts">
import type { operations } from '#shared/types/api/v1'
import { KUN_TOPIC_CATEGORY } from '~/constants/topic'
import { KUN_CATEGORY_DESCRIPTION_MAP } from '~/constants/category'

const route = useRoute()

const categoryName = computed(() => {
  return (route.params as { name: string }).name
})

type TopicCategory = NonNullable<
  NonNullable<operations['listSections']['parameters']['query']>['category']
>

const isTopicCategory = (value: string): value is TopicCategory =>
  value === 'galgame' || value === 'technique' || value === 'others'

const { data } = await useApi(
  () => `sections:${categoryName.value}`,
  (api, { signal }) =>
    api.GET('/sections', {
      params: {
        query: isTopicCategory(categoryName.value)
          ? { category: categoryName.value }
          : {}
      },
      signal
    })
)

useKunSeoMeta({
  title: KUN_TOPIC_CATEGORY[categoryName.value],
  description: KUN_CATEGORY_DESCRIPTION_MAP[categoryName.value]
})
</script>

<template>
  <CategoryContainer
    v-if="data && isTopicCategory(categoryName)"
    :sections="data.items"
    :category-name="categoryName"
  />
  <KunNull
    v-else-if="!isTopicCategory(categoryName)"
    description="没有这个分类"
  />
</template>
