<script setup lang="ts">
import type { TopicSummary } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  topic: TopicSummary
}>()

const author = computed(() => toKunUser(props.topic.author))
const actionsCount = computed(
  () => props.topic.reply_count + props.topic.comment_count
)
</script>

<template>
  <NuxtLink
    :to="`/topic/${topic.id}`"
    class="group block space-y-2 py-4 first:pt-0 last:pb-0"
  >
    <h3
      class="group-hover:text-primary line-clamp-2 text-lg font-medium transition-colors"
    >
      {{ topic.title }}
    </h3>

    <TopicBadgeGroup
      :section="topic.sections"
      :has-best-answer="topic.has_best_answer"
      :mini-apps="topic.mini_apps"
      :is-n-s-f-w-topic="topic.is_nsfw"
    />

    <div class="text-default-600 flex flex-wrap items-center gap-2 text-sm">
      <KunAvatar :user="author" size="xs" :is-navigation="false" />
      <span>{{ author.name }}</span>
      <KunTime :time="topic.created_at" type="relative" />

      <div class="text-default-500 ml-2 flex items-center gap-3">
        <span class="flex items-center gap-1">
          <KunIcon class="size-4" name="lucide:eye" />
          {{ formatNumber(topic.view_count) }}
        </span>
        <span v-if="topic.like_count" class="flex items-center gap-1">
          <KunIcon class="size-4" name="lucide:thumbs-up" />
          {{ topic.like_count }}
        </span>
        <span v-if="actionsCount" class="flex items-center gap-1">
          <KunIcon class="size-4" name="carbon:reply" />
          {{ actionsCount }}
        </span>
      </div>
    </div>
  </NuxtLink>
</template>
