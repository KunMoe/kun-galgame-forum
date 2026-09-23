<script setup lang="ts">
import type { CommentSearchHit } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  comment: CommentSearchHit
  keywords?: string
}>()

const author = computed(() => toKunUser(props.comment.author))
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    :to="commentPermalink(`/topic/${comment.topic_id}`, Number(comment.id))"
    class-name="flex-col items-start w-full gap-1.5"
  >
    <div class="flex w-full items-baseline gap-2">
      <KunIcon
        class="text-primary size-3.5 shrink-0 self-center"
        name="uil:comment-dots"
      />
      <span class="min-w-0 flex-1 truncate text-sm font-medium">
        <SearchHighlight :text="comment.topic_title" :keywords="keywords" />
      </span>
    </div>

    <p
      class="border-primary text-default-700 line-clamp-2 w-full border-l-2 pl-2 text-sm"
    >
      <SearchHighlight :text="comment.excerpt" :keywords="keywords" />
    </p>

    <div class="text-default-500 flex w-full items-center gap-1.5 text-xs">
      <KunAvatar size="xs" :user="author" :is-navigation="false" />
      <span class="truncate">{{ author.name }}</span>
      <span class="ml-auto shrink-0">
        <KunTime :time="comment.created_at" />
      </span>
    </div>
  </KunLink>
</template>
