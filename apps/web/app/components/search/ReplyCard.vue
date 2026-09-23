<script setup lang="ts">
import type { ReplySearchHit } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  reply: ReplySearchHit
  keywords?: string
}>()

const author = computed(() => toKunUser(props.reply.author))
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    :to="replyPermalink(`/topic/${reply.topic_id}`, reply.floor)"
    class-name="flex-col items-start w-full gap-1.5"
  >
    <div class="flex w-full items-baseline gap-2">
      <KunIcon
        class="text-primary size-3.5 shrink-0 self-center"
        name="carbon:reply"
      />
      <span class="min-w-0 flex-1 truncate text-sm font-medium">
        <SearchHighlight :text="reply.topic_title" :keywords="keywords" />
      </span>
      <span class="text-default-400 shrink-0 text-xs tabular-nums">
        #{{ reply.floor }}
      </span>
    </div>

    <p
      v-if="reply.excerpt"
      class="text-default-700 line-clamp-2 w-full text-sm"
    >
      <SearchHighlight :text="reply.excerpt" :keywords="keywords" />
    </p>

    <div class="text-default-500 flex w-full items-center gap-1.5 text-xs">
      <KunAvatar size="xs" :user="author" :is-navigation="false" />
      <span class="truncate">{{ author.name }}</span>
      <span class="ml-auto shrink-0">
        <KunTime :time="reply.created_at" />
      </span>
    </div>
  </KunLink>
</template>
