<script setup lang="ts">
const route = useRoute()
const { sectionOf } = useTopicSections()

const section = computed(() =>
  sectionOf((route.params as { section: string }).section)
)

useKunSeoMeta({
  title: section.value
    ? `${section.value.group_label} - ${section.value.label}`
    : '话题分区',
  description: section.value?.description ?? ''
})
</script>

<template>
  <SectionContainer v-if="section" :section="section" />
  <KunNull v-else description="没有这个版块" />
</template>
