<script setup lang="ts">
const emit = defineEmits<{ reveal: [] }>()
</script>

<template>
  <KunCard :is-hoverable="false" :is-transparent="false">
    <div class="flex flex-col items-start gap-3">
      <div class="flex items-center gap-2">
        <KunIcon name="lucide:lightbulb" class="text-warning size-5" />
        <p class="text-foreground font-medium">
          该游戏自带路线提示功能，建议先在游戏里打开
        </p>
      </div>
      <p class="text-default-500 text-sm">
        游戏内的提示通常比攻略更不容易剧透。如果仍想按攻略选择选项，可以继续查看。
      </p>
      <KunButton variant="flat" @click="emit('reveal')">仍然查看攻略</KunButton>
    </div>
  </KunCard>
</template>
