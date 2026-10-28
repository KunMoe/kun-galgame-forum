<script setup lang="ts">
defineProps<{
  user: {
    id: number
    name: string
    avatar: string
    moemoepoint: number
  }
  topicId: string
}>()
</script>

<template>
  <aside
    class="scrollbar-hide sticky top-20 hidden max-h-[calc(100dvh-6rem)] w-52 shrink-0 flex-col items-center gap-6 overflow-y-auto xl:flex"
  >
    <TopicDetailAuthorCard :user="user" />
    <TopicDetailAuthorTopics
      :user-id="user.id"
      :user-name="user.name"
      :topic-id="topicId"
    />
  </aside>
</template>
