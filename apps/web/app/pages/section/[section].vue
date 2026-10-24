<script setup lang="ts">
const route = useRoute()
const { sectionOf } = useTopicSections()

const key = computed(() => (route.params as { section: string }).section)
const section = computed(() => sectionOf(key.value))

useKunSeoMeta({
  title: section.value
    ? `${section.value.group_label} - ${section.value.label}`
    : '话题分区',
  description: section.value?.description ?? ''
})
</script>

<template>
  <TopicShell :active="key">
    <SectionContainer v-if="section" :section="section" />
    <KunNull v-else description="没有这个版块" />
  </TopicShell>
</template>
