<script setup lang="ts">
const props = defineProps<{ starred: boolean }>()
const emit = defineEmits<{ action: [key: string] }>()

const actions = computed(() => [
  {
    key: 'star',
    label: props.starred ? '取消收藏' : '收藏',
    icon: 'lucide:star'
  },
  { key: 'rename', label: '重命名', icon: 'lucide:pencil' },
  {
    key: 'delete',
    label: '删除',
    icon: 'lucide:trash-2',
    color: 'danger' as const
  }
])

const menu = useTemplateRef<{ close: () => void }>('menu')
const pick = (key: string) => {
  menu.value?.close()
  emit('action', key)
}
</script>

<template>
  <KunPopover
    ref="menu"
    position="bottom-end"
    inner-class="p-1 min-w-32"
    aria-label="对话操作"
  >
    <template #trigger>
      <KunButton variant="light" size="sm" is-icon-only aria-label="对话操作">
        <KunIcon name="lucide:ellipsis" class="size-4" />
      </KunButton>
    </template>
    <div class="flex flex-col">
      <button
        v-for="a in actions"
        :key="a.key"
        type="button"
        :class="
          cn(
            'hover:bg-default-100 rounded-kun-md flex w-full items-center gap-2 px-2 py-2 text-left text-sm',
            a.color === 'danger' && 'text-danger'
          )
        "
        @click="pick(a.key)"
      >
        <KunIcon :name="a.icon" class="size-4" />{{ a.label }}
      </button>
    </div>
  </KunPopover>
</template>
