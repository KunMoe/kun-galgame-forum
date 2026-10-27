<script setup lang="ts">
withDefaults(defineProps<{ to?: string }>(), {
  to: 'https://www.letmoe.com'
})
</script>

<template>
  <KunLink
    :to="to"
    target="_blank"
    rel="noopener noreferrer"
    size="sm"
    class-name="inline"
  >
    <slot>LetMoe·一起萌</slot>
  </KunLink>
</template>
