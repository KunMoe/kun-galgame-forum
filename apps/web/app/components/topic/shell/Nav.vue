<script setup lang="ts">
defineProps<{
  active: string
}>()

const { groups } = useTopicSections()
</script>

<template>
  <nav class="space-y-4 pb-4" aria-label="话题分区">
    <div class="space-y-1">
      <KunNavItem
        label="全部话题"
        icon="lucide:library-big"
        href="/topic"
        :current="active === ''"
        class-name="w-full justify-start"
      />
      <KunButton href="/edit/topic" color="primary" full-width>
        <KunIcon name="lucide:pencil" class="text-inherit" />
        发布话题
      </KunButton>
    </div>

    <div v-for="group in groups" :key="group.key" class="space-y-1">
      <p class="text-default-500 px-3 text-xs font-medium">
        {{ group.label }}
      </p>
      <KunNavItem
        v-for="section in group.sections"
        :key="section.section"
        :label="section.label"
        :icon="section.icon_name"
        :href="`/section/${section.section}`"
        :current="active === section.section"
        :color="section.color"
        :class-name="
          cn('w-full justify-start', section.is_archived && 'opacity-60')
        "
      />
    </div>
  </nav>
</template>
