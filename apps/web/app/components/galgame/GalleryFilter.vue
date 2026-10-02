<script setup lang="ts">
interface SourceOption {
  key: string
  label: string
  total: number
  on: boolean
}

const props = defineProps<{
  showLegend: boolean
  hiddenCount: number
  sources: SourceOption[]
}>()

const emit = defineEmits<{ toggleSource: [key: string] }>()

const showSources = computed(() => props.sources.length > 1)
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
      <div v-if="showSources" class="space-y-2">
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

      <KunDivider v-if="showSources && showLegend" />

      <p v-if="showLegend" class="text-default-400 text-xs leading-relaxed">
        缩略图描边:<span class="text-warning-500">外圈 = 色情</span
        >,颜色越深级别越高。
      </p>
    </div>
  </KunPopover>
</template>
