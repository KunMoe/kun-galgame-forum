<template>
  <div class="space-y-3" aria-hidden="true">
    <KunSkeleton height="1.5rem" width="40%" rounded="md" />
    <KunSkeleton v-for="n in 3" :key="n" height="6rem" rounded="lg" />
  </div>
</template>
