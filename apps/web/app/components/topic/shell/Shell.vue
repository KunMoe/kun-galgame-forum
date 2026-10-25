<script setup lang="ts">
defineProps<{
  active: string
}>()
</script>

<template>
  <div
    class="grid grid-cols-1 items-start gap-6 lg:grid-cols-[13rem_minmax(0,1fr)] xl:grid-cols-[13rem_minmax(0,1fr)_20rem]"
  >
    <!-- Columns are placed explicitly: the centre column's list takes a
         top-level await, and an auto-placed aside slid into its track while
         that was outstanding (the home page hit this first). -->
    <aside
      class="sticky top-20 hidden max-h-[calc(100dvh-6rem)] self-start overflow-y-auto lg:col-start-1 lg:block"
    >
      <TopicShellNav :active="active" />
    </aside>

    <div class="min-w-0 space-y-4 lg:col-start-2">
      <TopicShellPills :active="active" class="lg:hidden" />
      <slot />
    </div>

    <aside
      class="sticky top-20 hidden h-[calc(100dvh-6rem)] flex-col self-start xl:col-start-3 xl:flex"
    >
      <div class="min-h-0 overflow-y-auto">
        <TopicShellAside />
      </div>
      <HomeFooter class="mt-auto shrink-0 pt-4" />
    </aside>
  </div>
</template>
