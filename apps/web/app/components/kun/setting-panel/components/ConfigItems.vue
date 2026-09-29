<script setup lang="ts">
const {
  showKUNGalgamePageTransparency,
  showKUNGalgameBackgroundBrightness,
  showKUNGalgameRounded
} = storeToRefs(usePersistSettingsStore())

const roundedOptions = [
  { value: 'none', label: '直角' },
  { value: 'sm', label: '小' },
  { value: 'md', label: '中' },
  { value: 'lg', label: '大' }
] as const

watch(
  () => showKUNGalgamePageTransparency.value,
  debounce(() => {
    usePersistSettingsStore().setKUNGalgameTransparency(
      showKUNGalgamePageTransparency.value
    )
  }, 300)
)

watch(
  () => showKUNGalgameBackgroundBrightness.value,
  debounce(() => {
    usePersistSettingsStore().setKUNGalgameBackgroundBrightness(
      showKUNGalgameBackgroundBrightness.value
    )
  }, 300)
)
</script>

<template>
  <div class="space-y-5">
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <div class="text-default-700 flex items-center gap-2 font-medium">
          <KunIcon class="text-primary" name="mdi:circle-transparent" />
          <span>页面透明度</span>
        </div>
        <span class="text-default-500 text-sm tabular-nums">
          {{ showKUNGalgamePageTransparency }}%
        </span>
      </div>
      <KunSlider
        :min="10"
        :max="90"
        :step="1"
        v-model="showKUNGalgamePageTransparency"
      />
    </div>

    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <div class="text-default-700 flex items-center gap-2 font-medium">
          <KunIcon class="text-primary" name="lucide:lightbulb" />
          <span>背景亮度</span>
        </div>
        <span class="text-default-500 text-sm tabular-nums">
          {{ showKUNGalgameBackgroundBrightness }}%
        </span>
      </div>
      <KunSlider
        :min="10"
        :max="100"
        :step="1"
        v-model="showKUNGalgameBackgroundBrightness"
      />
    </div>

    <div class="space-y-2">
      <div class="text-default-700 flex items-center gap-2 font-medium">
        <KunIcon class="text-primary" name="tabler:border-radius" />
        <span>全局圆角</span>
      </div>
      <div class="grid grid-cols-4 gap-2">
        <KunButton
          v-for="opt in roundedOptions"
          :key="opt.value"
          size="sm"
          :variant="showKUNGalgameRounded === opt.value ? 'solid' : 'flat'"
          :color="showKUNGalgameRounded === opt.value ? 'primary' : 'default'"
          @click="usePersistSettingsStore().setKUNGalgameRounded(opt.value)"
        >
          {{ opt.label }}
        </KunButton>
      </div>
    </div>
  </div>
</template>
