<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="false"
    content-class="space-y-2"
  >
    <div class="flex items-center gap-2">
      <KunSkeleton height="1.5rem" width="4.5rem" rounded="full" />
      <KunSkeleton height="1.5rem" width="4rem" rounded="full" />
      <KunSkeleton variant="text" width="3.5rem" class-name="text-sm" />
    </div>
    <KunSkeleton variant="text" width="85%" />
    <div class="flex justify-between gap-2">
      <KunSkeleton variant="text" width="7rem" class-name="text-sm" />
      <KunSkeleton variant="text" width="5rem" class-name="text-sm" />
    </div>
  </KunCard>
</template>
