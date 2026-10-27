<template>
  <div class="space-y-10">
    <div class="flex items-start gap-4 sm:gap-6">
      <div class="w-24 shrink-0 sm:w-36">
        <KunSkeleton height="auto" rounded="lg" class-name="aspect-[3/4]" />
      </div>
      <div class="flex min-w-0 flex-1 flex-col gap-3">
        <KunSkeleton
          variant="text"
          width="60%"
          class-name="text-xl font-bold sm:text-3xl"
        />
        <KunSkeleton variant="text" width="35%" class-name="text-sm" />
        <KunSkeleton variant="text" width="80%" class-name="text-sm" />
      </div>
    </div>
    <div class="space-y-3">
      <KunSkeleton height="2.5rem" width="50%" rounded="lg" />
      <KunSkeleton height="20rem" rounded="lg" />
    </div>
  </div>
</template>
