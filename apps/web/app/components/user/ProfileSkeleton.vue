<template>
  <div class="space-y-4" aria-hidden="true">
    <KunCard :is-hoverable="false">
      <div class="flex items-start gap-4">
        <KunSkeleton variant="circle" width="6rem" height="6rem" />
        <div class="min-w-0 flex-1">
          <KunSkeleton variant="text" width="10rem" class-name="text-2xl" />
          <KunSkeleton
            height="1.5rem"
            width="4rem"
            rounded="full"
            class-name="mt-2"
          />
          <KunSkeleton variant="text" width="70%" class-name="mt-2 text-sm" />
        </div>
      </div>
      <div class="mt-5 flex flex-wrap gap-x-8 gap-y-3">
        <div v-for="n in 5" :key="n" class="min-w-14">
          <KunSkeleton variant="text" width="2.5rem" class-name="text-xl" />
          <KunSkeleton variant="text" width="3rem" class-name="text-xs" />
        </div>
      </div>
    </KunCard>

    <div
      class="grid grid-cols-1 items-start gap-4 sm:grid-cols-[auto_minmax(0,1fr)]"
    >
      <KunSkeleton height="2rem" rounded="lg" class-name="sm:hidden" />
      <div class="hidden w-28 space-y-3 sm:block">
        <KunSkeleton v-for="n in 6" :key="n" height="1.75rem" rounded="md" />
      </div>
      <UserListSkeleton :rows="4" />
    </div>
  </div>
</template>
