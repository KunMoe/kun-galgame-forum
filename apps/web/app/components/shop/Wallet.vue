<script setup lang="ts">
defineProps<{ balance: number | null }>()

const { showKUNGalgameMoemoepointLog } = storeToRefs(useTempSettingStore())
</script>

<template>
  <KunCard padding="md" class-name="sm:min-w-72">
    <div class="flex items-center gap-4">
      <div
        class="bg-secondary-100 text-secondary-600 flex size-12 shrink-0 items-center justify-center rounded-2xl"
      >
        <KunIcon name="lucide:lollipop" class="size-6" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-default-500 text-xs">我的萌萌点</p>
        <p
          class="text-foreground mt-1 text-3xl leading-none font-bold tracking-tight tabular-nums"
        >
          {{ balance ?? '—' }}
        </p>
      </div>
      <KunButton
        size="sm"
        variant="light"
        color="secondary"
        class-name="self-start"
        @click="showKUNGalgameMoemoepointLog = true"
      >
        明细
        <KunIcon name="lucide:chevron-right" class="size-4" />
      </KunButton>
    </div>
    <p class="text-default-400 text-xs leading-relaxed">
      每日签到、发布内容、被点赞和收藏都能攒萌萌点，
      <NuxtLink to="/point" class="text-primary hover:underline">
        怎么赚、怎么花
      </NuxtLink>
    </p>
  </KunCard>
</template>
