<script setup lang="ts">
const {
  showKUNGalgamePageTransparency,
  showKUNGalgameBackgroundBrightness,
  showKUNGalgameRounded,
  showKUNGalgamePhoneColumns
} = storeToRefs(usePersistSettingsStore())

const roundedOptions = [
  { value: 'none', label: '直角' },
  { value: 'sm', label: '小' },
  { value: 'md', label: '中' },
  { value: 'lg', label: '大' }
] as const

const phoneColumnOptions = [
  { value: 2, label: '2 张' },
  { value: 3, label: '3 张' }
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

    <div class="space-y-2">
      <div class="text-default-700 flex items-center gap-2 font-medium">
        <KunIcon class="text-primary" name="lucide:layout-grid" />
        <span>手机上每行的游戏卡片</span>
      </div>
      <div class="grid grid-cols-2 gap-2">
        <KunButton
          v-for="opt in phoneColumnOptions"
          :key="opt.value"
          size="sm"
          :variant="showKUNGalgamePhoneColumns === opt.value ? 'solid' : 'flat'"
          :color="
            showKUNGalgamePhoneColumns === opt.value ? 'primary' : 'default'
          "
          @click="showKUNGalgamePhoneColumns = opt.value"
        >
          {{ opt.label }}
        </KunButton>
      </div>
    </div>
  </div>
</template>
