<script setup lang="ts">
import { tagPathsUnder } from '~/utils/galgame/tagTree'

const props = defineProps<{ tagId: number }>()

const { nodes, failed } = useTagDescendants(() => props.tagId)

const rows = computed(() => {
  const list = nodes.value ?? []
  const paths = tagPathsUnder(String(props.tagId), list)
  return list.map((node) => ({
    node,
    name: catalogVocabularyName(node),
    path: (paths.get(node.id) ?? []).map(catalogVocabularyName).join(' › ')
  }))
})

const keyword = ref('')
const matched = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) {
    return rows.value
  }
  return rows.value.filter(
    (row) =>
      row.name.toLowerCase().includes(q) ||
      row.node.display_name.toLowerCase().includes(q)
  )
})
</script>

<template>
  <div class="space-y-2">
    <p v-if="failed" class="text-default-400 text-sm">
      子标签列表暂时读取失败, 请稍后再试
    </p>

    <p
      v-else-if="!nodes"
      class="text-default-400 flex items-center gap-1.5 text-sm"
    >
      <KunIcon name="lucide:loader-circle" class="size-4 animate-spin" />
      正在读取子标签
    </p>

    <template v-else>
      <KunInput
        v-if="nodes.length > 8"
        v-model="keyword"
        size="sm"
        placeholder="在子标签中查找"
        is-clearable
      >
        <template #prefix>
          <KunIcon name="lucide:search" class="text-default-400 size-4" />
        </template>
      </KunInput>

      <KunScrollShadow
        axis="vertical"
        scrollbar="thin"
        class-name="max-h-64"
        content-class="gap-0"
      >
        <div
          v-for="row in matched"
          :key="row.node.id"
          class="border-default-100 flex min-w-0 flex-col border-b py-1.5 last:border-b-0"
        >
          <span
            :class="
              cn(
                'text-sm',
                row.node.is_sexual ? 'text-danger-600' : 'text-default-700'
              )
            "
          >
            {{ row.name }}
          </span>
          <span v-if="row.path" class="text-default-400 truncate text-xs">
            {{ row.path }}
          </span>
        </div>

        <p
          v-if="!matched.length"
          class="text-default-400 py-3 text-center text-sm"
        >
          没有匹配的子标签
        </p>
      </KunScrollShadow>

      <p class="text-default-400 text-xs">
        共 {{ nodes.length.toLocaleString('en-US') }} 个子标签<template
          v-if="keyword.trim()"
          >, 匹配 {{ matched.length }} 个</template
        >
      </p>
    </template>
  </div>
</template>
