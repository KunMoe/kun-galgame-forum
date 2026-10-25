<script setup lang="ts">
const route = useRoute()
const { sectionOf } = useTopicSections()

const requested = computed(() =>
  typeof route.query.section === 'string' ? route.query.section : ''
)
const active = computed(
  () => sectionOf(requested.value)?.section ?? requested.value
)

watch(
  active,
  async (key) => {
    if (key !== requested.value) {
      await navigateTo(
        { query: { ...route.query, section: key } },
        { replace: true, redirectCode: 301 }
      )
    }
  },
  { immediate: true }
)
</script>

<template>
  <TopicShell :active="active">
    <TopicContainer :section-key="active" />
  </TopicShell>
</template>
