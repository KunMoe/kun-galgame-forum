<script setup lang="ts">
defineProps<{ balance: number | null }>()

const { showKUNGalgameMoemoepointLog } = storeToRefs(useTempSettingStore())
</script>

<template>
  <div class="space-y-1 sm:text-right">
    <p class="text-default-500 text-sm">我的萌萌点</p>
    <p
      class="text-foreground text-4xl leading-none font-semibold tracking-tight tabular-nums"
    >
      {{ balance ?? '—' }}
    </p>
    <div class="-mx-2 flex sm:justify-end">
      <KunButton
        size="sm"
        variant="light"
        @click="showKUNGalgameMoemoepointLog = true"
      >
        收支明细
      </KunButton>
      <KunButton size="sm" variant="light" href="/point">
        怎么赚萌萌点
      </KunButton>
    </div>
  </div>
</template>
