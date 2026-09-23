<script setup lang="ts">
import type { Ref } from 'vue'

interface SourceOption {
  key: string
  label: string
  total: number
  on: boolean
}

const props = defineProps<{
  showNsfw: boolean
  hiddenCount: number
  sexualCounts: Record<number, number>
  sources: SourceOption[]
}>()

const emit = defineEmits<{ toggleSource: [key: string] }>()

const { showKUNGalgameGallerySexualLevels: sexualLevels } = storeToRefs(
  usePersistSettingsStore()
)

const LEVELS = [
  { value: 1, label: '轻' },
  { value: 2, label: '中' },
  { value: 3, label: '高' }
]

const sexualShown = computed(() =>
  LEVELS.filter((lv) => (props.sexualCounts[lv.value] ?? 0) > 0)
)

const showSources = computed(() => props.sources.length > 1)

const toggle = (arr: Ref<number[]>, level: number) => {
  arr.value = arr.value.includes(level)
    ? arr.value.filter((l) => l !== level)
    : [...arr.value, level]
}

const toggleSexual = (level: number) => toggle(sexualLevels, level)
</script>

<template>
  <KunPopover position="bottom-end" inner-class="w-72 p-3">
    <template #trigger>
      <KunButton variant="flat" size="sm">
        <KunIcon name="lucide:funnel" />
        筛选
        <KunChip v-if="hiddenCount" size="sm" color="warning" variant="flat">
          隐藏 {{ hiddenCount }}
        </KunChip>
      </KunButton>
    </template>

    <div class="space-y-4">
      <template v-if="showSources">
        <div class="space-y-2">
          <p class="text-default-700 text-sm font-medium">图片来源</p>
          <div class="flex flex-col gap-2">
            <KunCheckBox
              v-for="s in sources"
              :id="`gal-source-${s.key || 'unknown'}`"
              :key="s.key"
              type="single"
              :model-value="s.on"
              :label="`${s.label} · ${s.total} 张`"
              @update:model-value="() => emit('toggleSource', s.key)"
            />
          </div>
        </div>

        <KunDivider />
      </template>

      <div class="space-y-2">
        <p class="text-default-700 text-sm font-medium">色情评级</p>
        <p v-if="!sexualShown.length" class="text-default-400 text-xs">
          无色情评级图片
        </p>
        <p v-else-if="showNsfw" class="text-default-500 text-xs">
          NSFW 模式已显示全部色情内容。
        </p>
        <div v-else class="flex flex-col gap-2">
          <KunCheckBox
            v-for="lv in sexualShown"
            :id="`gal-sexual-${lv.value}`"
            :key="lv.value"
            type="single"
            :model-value="sexualLevels.includes(lv.value)"
            :label="`${lv.label} · ${sexualCounts[lv.value]} 张`"
            @update:model-value="() => toggleSexual(lv.value)"
          />
        </div>
      </div>

      <KunDivider />
      <p class="text-default-400 text-xs leading-relaxed">
        缩略图描边:<span class="text-warning-500">外圈 = 色情</span
        >,颜色越深级别越高。
      </p>
    </div>
  </KunPopover>
</template>
