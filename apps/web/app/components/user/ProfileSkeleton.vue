<script setup lang="ts">
const route = useRoute()
const tab = computed(() => {
  const path = route.path.replace(/\/+$/, '')
  if (path.endsWith('/library')) return 'library'
  return /^\/user\/\d+(\/activity)?$/.test(path) ? 'activity' : 'list'
})
</script>

<template>
  <div
    class="grid grid-cols-1 items-start gap-4 lg:grid-cols-[18rem_minmax(0,1fr)] lg:gap-6"
    aria-hidden="true"
  >
    <KunCard :is-hoverable="false" content-class="gap-4">
      <div class="flex items-start gap-4 lg:flex-col lg:gap-3">
        <KunSkeleton variant="circle" width="6rem" height="6rem" />
        <div class="min-w-0 flex-1 space-y-2">
          <KunSkeleton variant="text" width="10rem" class-name="text-2xl" />
          <KunSkeleton height="1.5rem" width="4rem" rounded="full" />
          <KunSkeleton variant="text" width="70%" class-name="text-sm" />
        </div>
      </div>
      <div
        class="grid grid-cols-4 gap-x-4 gap-y-3 sm:grid-cols-8 lg:grid-cols-2"
      >
        <div v-for="n in 8" :key="n">
          <KunSkeleton variant="text" width="2.5rem" class-name="text-xl" />
          <KunSkeleton variant="text" width="3rem" class-name="text-xs" />
        </div>
      </div>
    </KunCard>

    <div class="min-w-0 space-y-4">
      <KunSkeleton height="2.25rem" rounded="lg" />
      <UserLibrarySkeleton v-if="tab === 'library'" />
      <UserTimelineSkeleton v-else-if="tab === 'activity'" />
      <UserListSkeleton v-else :rows="4" />
    </div>
  </div>
</template>
