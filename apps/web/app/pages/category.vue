<script setup lang="ts">
const route = useRoute()
const { groups } = useTopicSections()

const group = computed(() =>
  groups.value.find(
    (g) => g.key === (route.params as { name?: string }).name
  )
)
</script>

<template>
  <div class="h-full justify-start space-y-3">
    <KunHeader
      v-if="group"
      :name="group.label"
      :description="group.sections.map((s) => s.label).join(' · ')"
    />

    <NuxtPage />
  </div>
</template>
