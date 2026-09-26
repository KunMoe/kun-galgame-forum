<script setup lang="ts">
defineProps<{
  userId: number | string
}>()
</script>

<template>
  <KunHoverCard group="user-hover-card" opaque :disabled="!Number(userId)">
    <template #trigger>
      <slot />
    </template>
    <template #default="{ close }">
      <UserHoverCardPanel :user-id="userId" @navigate="close" />
    </template>
  </KunHoverCard>
</template>
