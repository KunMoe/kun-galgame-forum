<script setup lang="ts">
import type { TopicSummary } from '#shared/utils/api/schemas'
import type { operations } from '#shared/types/api/v1'
import { problemMessage } from '#shared/utils/api/message'
import { KUN_TOPIC_CATEGORY, KUN_TOPIC_SECTION } from '~/constants/topic'
import { KUN_TOPIC_SECTION_DESCRIPTION_MAP } from '~/constants/section'
import { useCursorList } from '~/composables/useCursorList'

type SectionSlug = NonNullable<
  NonNullable<operations['listTopics']['parameters']['query']>['section']
>

const props = defineProps<{
  section: SectionSlug
}>()

const categoryMap: Record<string, string> = {
  g: 'galgame',
  t: 'technique',
  o: 'others'
}
const category = computed(
  () => KUN_TOPIC_CATEGORY[categoryMap[props.section[0]!]!]!
)

const { allowsNsfw: includeNsfw } = useContentStance()

const { items, hasMore, problem, status, loadingMore, loadMore, refresh } =
  await useCursorList<TopicSummary>(
    () =>
      `section-topics:${props.section}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
    (api, cursor, { signal }) =>
      api.GET('/topics', {
        params: {
          query: {
            section: props.section,
            sort: 'created_desc',
            limit: 30,
            include_nsfw: includeNsfw.value,
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )
</script>

<template>
  <div class="space-y-6">
    <KunHeader :description="KUN_TOPIC_SECTION_DESCRIPTION_MAP[section]">
      <template #title>
        <div class="flex items-center gap-2">
          <KunLink
            underline="hover"
            :to="`/category/${categoryMap[props.section[0]!]}`"
            class-name="text-2xl font-medium"
          >
            {{ category }}
          </KunLink>
          /
          <span class="text-lg">{{ KUN_TOPIC_SECTION[section] }}</span>
        </div>
      </template>
    </KunHeader>

    <template v-if="problem">
      <KunNull :description="problemMessage(problem)" />
      <div class="flex justify-center">
        <KunButton variant="flat" size="sm" @click="() => refresh()">
          重试
        </KunButton>
      </div>
    </template>

    <template v-else>
      <KunLoading :loading="status === 'pending'">
        <div class="divide-default-200/60 divide-y">
          <TopicCard v-for="topic in items" :key="topic.id" :topic="topic" />
        </div>
      </KunLoading>

      <KunNull
        v-if="status !== 'pending' && !items.length"
        description="这个版块还没有话题"
      />

      <div v-if="items.length" class="flex justify-center pt-4">
        <KunButton
          v-if="hasMore"
          variant="light"
          :loading="loadingMore"
          @click="loadMore"
        >
          加载更多
        </KunButton>
        <span v-else class="text-default-400 text-sm">没有更多话题了</span>
      </div>
    </template>
  </div>
</template>
