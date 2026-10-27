<template>
  <div
    class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4"
    aria-hidden="true"
  >
    <div v-for="n in 8" :key="n" class="space-y-2">
      <KunSkeleton height="auto" rounded="lg" class-name="aspect-video" />
      <KunSkeleton variant="text" width="80%" class-name="text-sm" />
    </div>
  </div>
</template>
