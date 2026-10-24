<script setup lang="ts">
const route = useRoute()
const { groups } = useTopicSections()

const name = computed(() => (route.params as { name: string }).name)
if (name.value === 'others') {
  await navigateTo('/category/community', { redirectCode: 301 })
}
const group = computed(() => groups.value.find((g) => g.key === name.value))

useKunSeoMeta({
  title: group.value?.label ?? '话题分类',
  description: group.value?.sections.map((s) => s.label).join(', ') ?? ''
})
</script>

<template>
  <CategoryContainer v-if="group" :sections="group.sections" />
  <KunNull v-else description="没有这个分类" />
</template>
