<script setup lang="ts">
import type { Section, TopicSummary } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { useCursorList } from '~/composables/useCursorList'

const props = defineProps<{
  section: Section
}>()

const { allowsNsfw: includeNsfw } = useContentStance()

const { items, hasMore, problem, status, loadingMore, loadMore, refresh } =
  await useCursorList<TopicSummary>(
    () =>
      `section-topics:${props.section.section}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
    (api, cursor, { signal }) =>
      api.GET('/topics', {
        params: {
          query: {
            section: props.section.section,
            sort: 'bumped_desc',
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
    <KunHeader :description="section.description">
      <template #title>
        <div class="flex items-center gap-2">
          <KunLink
            underline="hover"
            :to="`/category/${section.group}`"
            class-name="text-2xl font-medium"
          >
            {{ section.group_label }}
          </KunLink>
          /
          <span class="text-lg">{{ section.label }}</span>
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
