<template>
  <div
    class="grid grid-cols-1 items-start gap-6 lg:grid-cols-[minmax(0,1fr)_22rem] xl:grid-cols-[15rem_minmax(0,1fr)_22rem]"
    aria-hidden="true"
  >
    <div class="hidden space-y-4 xl:block">
      <div v-for="n in 5" :key="n" class="space-y-2 p-2.5">
        <KunSkeleton variant="text" width="80%" />
        <KunSkeleton variant="text" width="50%" class-name="text-sm" />
      </div>
    </div>

    <div class="space-y-6">
      <div class="space-y-3">
        <KunSkeleton variant="text" width="14rem" class-name="text-2xl" />
        <KunSkeleton variant="text" width="60%" />
        <KunSkeleton variant="text" width="20rem" />
      </div>
      <GalgameCollectionEntryListSkeleton />
    </div>

    <div class="space-y-3">
      <KunSkeleton variant="text" width="6rem" class-name="text-2xl" />
      <KunSkeleton variant="rect" height="8rem" class-name="rounded-lg" />
    </div>
  </div>
</template>
