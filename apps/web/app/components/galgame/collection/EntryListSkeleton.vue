<template>
  <div class="space-y-8" aria-hidden="true">
    <div v-for="n in 4" :key="n" class="flex gap-4">
      <KunSkeleton
        variant="rect"
        class-name="w-24 shrink-0 rounded-md sm:w-32"
        style="aspect-ratio: 5 / 7"
      />
      <div class="flex-1 space-y-2">
        <KunSkeleton variant="text" width="70%" class-name="text-lg" />
        <KunSkeleton variant="text" width="40%" />
        <KunSkeleton variant="text" width="55%" />
        <KunSkeleton variant="text" width="30%" />
      </div>
    </div>
  </div>
</template>
