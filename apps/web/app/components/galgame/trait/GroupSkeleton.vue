<template>
  <div class="space-y-4" aria-hidden="true">
    <div class="flex flex-wrap gap-1.5">
      <KunSkeleton
        v-for="n in 8"
        :key="n"
        height="1.75rem"
        :width="`${4 + (n % 3)}rem`"
        rounded="full"
      />
    </div>
    <div
      class="grid grid-cols-1 items-start gap-3 md:grid-cols-2 xl:grid-cols-3"
    >
      <div
        v-for="n in 6"
        :key="n"
        class="border-default-200 space-y-2 rounded-lg border p-3"
      >
        <KunSkeleton height="1.75rem" width="6rem" rounded="full" />
        <div class="flex flex-wrap gap-1.5">
          <KunSkeleton
            v-for="m in 5"
            :key="m"
            height="1.75rem"
            :width="`${3.5 + (m % 3)}rem`"
            rounded="full"
          />
        </div>
      </div>
    </div>
  </div>
</template>
