<script setup lang="ts">
const props = defineProps<{
  ids: string[]
  current: string
  disabled?: boolean
}>()
const emit = defineEmits<{ switch: [id: string] }>()

const index = computed(() => props.ids.indexOf(props.current))
const go = (step: number) => {
  const id = props.ids[index.value + step]
  if (id) emit('switch', id)
}
</script>

<template>
  <div
    v-if="ids.length > 1"
    class="text-default-500 flex items-center text-xs tabular-nums"
  >
    <AgentIconAction
      icon="lucide:chevron-left"
      label="上一个版本"
      :disabled="disabled || index <= 0"
      @click="go(-1)"
    />
    <span class="min-w-8 text-center">{{ index + 1 }}/{{ ids.length }}</span>
    <AgentIconAction
      icon="lucide:chevron-right"
      label="下一个版本"
      :disabled="disabled || index >= ids.length - 1"
      @click="go(1)"
    />
  </div>
</template>
