<script setup lang="ts">
import type { Topic } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

defineProps<{
  topic: Topic
}>()

const { id } = usePersistUserStore()
</script>

<template>
  <div class="mt-auto hidden items-center justify-between leading-none md:flex">
    <div class="flex items-center gap-1">
      <TopicFooterUpvote :topic="topic" />

      <TopicFooterFavorite :topic="topic" />

      <TopicFooterSubscribe :topic-id="topic.id" />

      <TopicReactionTrigger />
    </div>

    <div class="flex items-center gap-1">
      <TopicFooterReply
        :target-user-name="toKunUser(topic.author).name"
        :target-user-id="Number(topic.author.id)"
        :target-floor="0"
      />

      <TopicFooterRewrite :topic="topic" />

      <KunPopover position="top-end">
        <template #trigger>
          <KunReaction :toggle="false" icon="lucide:ellipsis" label="更多" />
        </template>

        <div class="flex w-54 flex-col gap-2 p-2">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            class-name="w-full justify-start gap-2 whitespace-nowrap"
            @click="
              useKunCopy(
                `${topic.title}: https://www.kungal.com/topic/${topic.id}`
              )
            "
          >
            <KunIcon class-name="text-lg" name="lucide:share-2" />
            分享
          </KunButton>
          <TopicFooterHide v-if="id" :topic="topic" />
          <ReportButton
            v-if="Number(topic.author.id) !== id"
            menu
            subject-kind="forum_topic"
            :subject-id="Number(topic.id)"
            :snapshot="topic.title"
            :subject-url="`${kungal.domain.main}/topic/${topic.id}`"
          />
        </div>
      </KunPopover>
    </div>
  </div>
</template>
