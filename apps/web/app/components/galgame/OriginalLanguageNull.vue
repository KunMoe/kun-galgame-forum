<script setup lang="ts">
defineProps<{
  description: string
}>()

const { showKUNGalgameAllOriginalLanguages } = storeToRefs(
  usePersistSettingsStore()
)
</script>

<template>
  <div class="flex flex-col items-center gap-3">
    <KunNull :description="description" />

    <template v-if="!showKUNGalgameAllOriginalLanguages">
      <p class="text-default-500 text-center text-sm">
        列表默认只显示原语言为日语或中文的 Galgame
      </p>
      <KunButton
        variant="flat"
        @click="showKUNGalgameAllOriginalLanguages = true"
      >
        显示全部原语言的 Galgame
      </KunButton>
    </template>
  </div>
</template>
