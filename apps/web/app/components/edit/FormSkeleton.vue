<template>
  <div class="space-y-6">
    <div class="space-y-2">
      <KunSkeleton variant="text" width="3rem" class-name="text-sm" />
      <KunSkeleton height="2.5rem" rounded="lg" />
    </div>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <KunSkeleton v-for="n in 4" :key="n" height="3.5rem" rounded="lg" />
    </div>
    <KunSkeleton height="20rem" rounded="lg" />
    <div class="flex justify-end">
      <KunSkeleton height="2.5rem" width="6rem" rounded="lg" />
    </div>
  </div>
</template>
