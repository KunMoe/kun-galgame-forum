<template>
  <div class="grid grid-cols-1 gap-3 lg:grid-cols-3" aria-hidden="true">
    <div class="min-w-0 space-y-3 lg:col-span-2">
      <KunCard
        :is-transparent="false"
        :is-hoverable="false"
        class-name="p-6"
        content-class="space-y-6"
      >
        <div class="flex items-start gap-6">
          <KunSkeleton width="5rem" height="5rem" rounded="lg" />
          <div class="flex-1 space-y-3">
            <KunSkeleton variant="text" width="50%" class-name="text-3xl" />
            <KunSkeleton variant="text" width="14rem" class-name="text-sm" />
          </div>
        </div>
        <div>
          <KunSkeleton
            v-for="n in 3"
            :key="n"
            variant="text"
            :width="n === 3 ? '60%' : '100%'"
            class-name="text-lg leading-relaxed"
          />
        </div>
        <KunSkeleton height="10rem" rounded="lg" />
      </KunCard>
    </div>

    <div class="min-w-0 space-y-3">
      <KunSkeleton height="14rem" rounded="lg" />
      <KunSkeleton height="8rem" rounded="lg" />
    </div>
  </div>
</template>
