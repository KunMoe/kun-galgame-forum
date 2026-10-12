<script setup lang="ts">
const props = withDefaults(
  defineProps<{ tagId: number; count: number; preview?: number }>(),
  { preview: 12 }
)

const { nodes, failed } = useTagDescendants(() => props.tagId)

const total = computed(() => nodes.value?.length ?? props.count)
const shown = computed(() => nodes.value?.slice(0, props.preview) ?? [])
const rest = computed(() => total.value - shown.value.length)
const expanded = ref(false)
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <p class="text-default-600 text-sm">
        将同时屏蔽
        <span class="text-danger-600 font-medium tabular-nums">
          {{ total.toLocaleString('en-US') }}
        </span>
        个子标签
      </p>
      <KunButton
        v-if="nodes && rest > 0"
        size="xs"
        variant="light"
        @click="expanded = !expanded"
      >
        {{ expanded ? '收起' : '查看全部' }}
      </KunButton>
    </div>

    <p v-if="failed" class="text-default-400 text-xs">
      子标签列表暂时读取失败, 屏蔽照常生效
    </p>

    <GalgameTagBlockDescendants v-else-if="nodes && expanded" :tag-id="tagId" />

    <div v-else-if="nodes" class="flex flex-wrap gap-1.5">
      <KunChip
        v-for="node in shown"
        :key="node.id"
        size="xs"
        variant="flat"
        :color="node.is_sexual ? 'danger' : 'default'"
      >
        {{ catalogVocabularyName(node) }}
      </KunChip>
      <KunChip v-if="rest > 0" size="xs" variant="bordered">
        +{{ rest.toLocaleString('en-US') }}
      </KunChip>
    </div>

    <p v-else class="text-default-400 flex items-center gap-1.5 text-xs">
      <KunIcon name="lucide:loader-circle" class="size-3.5 animate-spin" />
      正在读取子标签
    </p>
  </div>
</template>
