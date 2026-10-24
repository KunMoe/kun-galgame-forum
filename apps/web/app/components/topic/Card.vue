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

    <p
      v-if="topic.ai_summary"
      class="text-default-600 line-clamp-3 text-sm leading-relaxed"
    >
      <span
        class="bg-default-100 text-default-500 mr-1.5 rounded px-1.5 py-0.5 text-xs"
      >
        AI 摘要
      </span>
      {{ topic.ai_summary }}
    </p>

    <TopicBadgeGroup
      :section="topic.section"
      :has-best-answer="topic.has_best_answer"
      :mini-apps="topic.mini_apps"
      :is-n-s-f-w-topic="topic.is_nsfw"
    />

    <div class="text-default-600 flex flex-wrap items-center gap-2 text-sm">
      <KunAvatar :user="author" size="xs" :is-navigation="false" />
      <span>{{ author.name }}</span>
      <KunTime :time="topic.bumped_at" type="relative" />
      <span v-if="topic.edited_at" class="flex items-center gap-1">
        <KunIcon class="size-3.5" name="lucide:pencil" />
        <KunTime :time="topic.edited_at" type="relative" />
      </span>

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
