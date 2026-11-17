<script setup lang="ts">
defineProps<{ keywords: string }>()
</script>

<template>
  <div
    class="bg-primary/5 border-primary/20 rounded-kun-md flex flex-wrap items-center gap-x-3 gap-y-2 border px-3 py-2 text-sm"
  >
    <KunIcon name="lucide:sparkles" class="text-primary size-4 shrink-0" />
    <span class="text-default-600 min-w-0 flex-1">
      没找到想要的? 说说剧情, 角色或氛围, 让 AI 帮你找。
    </span>
    <KunButton
      :href="agentAskHref(keywords)"
      variant="flat"
      color="primary"
      size="sm"
    >
      用 AI 帮找
    </KunButton>
  </div>
</template>
