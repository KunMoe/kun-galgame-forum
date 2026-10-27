<template>
  <div class="flex gap-3 p-2" aria-hidden="true">
    <KunSkeleton variant="circle" width="3rem" height="3rem" />
    <div class="flex min-w-0 flex-1 flex-col justify-center">
      <div class="flex items-center justify-between">
        <KunSkeleton variant="text" width="6rem" />
        <KunSkeleton variant="text" width="3rem" class-name="text-sm" />
      </div>
      <KunSkeleton variant="text" width="80%" class-name="text-sm" />
    </div>
  </div>
</template>
