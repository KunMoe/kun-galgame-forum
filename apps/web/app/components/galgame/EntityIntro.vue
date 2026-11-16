<script setup lang="ts">
import { useResizeObserver } from '@vueuse/core'

const props = defineProps<{
  text: string
  note?: string
}>()

const paragraph = useTemplateRef('paragraph')
const isExpanded = ref(false)
const isOverflowing = ref(
  props.text.length > 100 || props.text.split('\n').length > 3
)

const measure = () => {
  const el = paragraph.value
  if (el && !isExpanded.value) {
    isOverflowing.value = el.scrollHeight > el.clientHeight + 1
  }
}

onMounted(measure)
useResizeObserver(paragraph, measure)
watch(
  () => props.text,
  () => nextTick(measure)
)
</script>

<template>
  <div class="space-y-1">
    <p
      ref="paragraph"
      :class="
        cn(
          'text-default-600 text-sm whitespace-pre-line sm:text-base',
          !isExpanded && 'line-clamp-3'
        )
      "
    >
      {{ text }}
    </p>
    <p v-if="note" class="text-default-400 text-xs">{{ note }}</p>
    <KunButton
      v-if="isExpanded || isOverflowing"
      variant="light"
      size="sm"
      color="primary"
      @click="isExpanded = !isExpanded"
    >
      <KunIcon
        :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
      />
      {{ isExpanded ? '收起简介' : '展开简介' }}
    </KunButton>
  </div>
</template>
