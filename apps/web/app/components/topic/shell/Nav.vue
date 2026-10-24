<script setup lang="ts">
defineProps<{
  active: string
}>()

const { sections } = useTopicSections()
</script>

<template>
  <nav class="space-y-1 pb-4" aria-label="话题分区">
    <KunButton href="/edit/topic" color="primary" full-width class-name="mb-3">
      <KunIcon name="lucide:pencil" class="text-inherit" />
      发布话题
    </KunButton>
    <KunNavItem
      label="全部话题"
      icon="lucide:library-big"
      href="/topic"
      :current="active === ''"
      class-name="w-full justify-start"
    />
    <KunNavItem
      v-for="section in sections"
      :key="section.section"
      :label="section.label"
      :icon="section.icon_name"
      :href="`/topic?section=${section.section}`"
      :current="active === section.section"
      :color="section.color"
      :class-name="
        cn('w-full justify-start', section.is_archived && 'opacity-60')
      "
    />
  </nav>
</template>
