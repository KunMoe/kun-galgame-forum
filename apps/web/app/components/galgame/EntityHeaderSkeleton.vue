<template>
  <div class="space-y-2" aria-hidden="true">
    <KunSkeleton
      variant="text"
      width="45%"
      class-name="text-2xl font-medium sm:text-3xl"
    />
    <KunSkeleton variant="text" width="80%" class-name="text-sm sm:text-base" />
    <div>
      <KunSkeleton variant="text" />
      <KunSkeleton variant="text" width="60%" />
    </div>
  </div>
</template>
