<script setup lang="ts">
import type { KunSelectValue } from '@kungal/ui-vue'
import { storeToRefs } from 'pinia'
import { fetchEntityFamily } from '~/utils/search/entities'

const props = withDefaults(
  defineProps<{
    family: 'company' | 'tag'
    label: string
    icon: string
    placeholder: string
    selectedIds: number[]
    /** Whatever the caller has already resolved names for; listed when idle. */
    selectedItems?: SearchEntityItem[]
    /** What still narrows the results on screen, most works first; listed when idle. */
    suggestions?: SearchEntityItem[]
    multiple?: boolean
    max?: number
  }>(),
  {
    selectedItems: () => [],
    suggestions: () => [],
    multiple: false,
    max: 0
  }
)

const emit = defineEmits<{ toggle: [item: SearchEntityItem] }>()

const query = ref('')
const found = ref<SearchEntityItem[]>([])
const loading = ref(false)
const failed = ref(false)
const api = useApiClient()
const { allowsNsfw } = useContentStance()
const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)

let latest = 0

const search = async (keywords: string) => {
  const current = ++latest
  query.value = keywords
  const q = keywords.trim()
  if (!q) {
    found.value = []
    failed.value = false
    loading.value = false
    return
  }
  loading.value = true
  const group = await fetchEntityFamily(
    api,
    props.family,
    q,
    1,
    20,
    allowsNsfw.value,
    showKUNGalgamePreferOriginalName.value
  )
  if (current !== latest) {
    return
  }
  failed.value = group.failed ?? false
  found.value = group.failed ? [] : group.items
  loading.value = false
}

const full = computed(
  () => props.multiple && !!props.max && props.selectedIds.length >= props.max
)

const pool = computed(() =>
  query.value.trim() ? found.value : props.suggestions
)

const known = computed(() => {
  const map = new Map<number, SearchEntityItem>()
  for (const item of [...pool.value, ...props.selectedItems]) {
    map.set(item.id, item)
  }
  return map
})

// Selected first and always present: a value with no option behind it renders
// as a bare id in the trigger and cannot be clicked off again.
const options = computed(() => [
  ...props.selectedIds.map((id) => ({
    value: id,
    label: known.value.get(id)?.name ?? `#${id}`,
    hint: '已选'
  })),
  ...pool.value
    .filter((item) => !props.selectedIds.includes(item.id))
    .map((item) => ({
      value: item.id,
      label: item.name,
      hint: item.work_count ? `${item.work_count} 部` : '',
      disabled: full.value
    }))
])

const pick = (value: KunSelectValue) => {
  const id = Number(value)
  const item = known.value.get(id) ?? {
    id,
    family: props.family,
    name: `#${id}`
  }
  emit('toggle', item)
}

const trigger = computed(() => {
  const n = props.selectedIds.length
  return {
    icon: n ? props.icon : 'lucide:plus',
    text: props.multiple && n ? `${props.label} ${n}` : `添加${props.label}`
  }
})

// The URL owns the selection: @set carries the pick up, the route comes back
// down as `selectedIds`. This listener only has to exist — Vue's defineModel
// switches to local state when no `update:modelValue` handler is attached, and
// a local copy is no longer the URL's.
const keepControlled = () => {}

const emptyText = computed(() =>
  query.value.trim()
    ? failed.value
      ? `${props.label}搜索没能完成, 请稍后重试`
      : `没有找到匹配的${props.label}`
    : `输入关键词以搜索${props.label}`
)
</script>

<template>
  <KunSelect
    :model-value="multiple ? null : (selectedIds[0] ?? null)"
    :options="options"
    :icon="trigger.icon"
    :placeholder="trigger.text"
    :aria-label="`按${label}筛选`"
    :full-width="false"
    :class-names="{ trigger: filterPillClass(selectedIds.length > 0) }"
    :search-placeholder="full ? `最多同时筛选 ${max} 个${label}` : placeholder"
    :no-result-text="emptyText"
    :loading="loading"
    :debounce="300"
    loading-text="正在检索鲲 Galgame 论坛数据…"
    popup-width="auto"
    rounded="full"
    size="sm"
    searchable
    manual-filter
    @search="search"
    @set="pick"
    @update:model-value="keepControlled"
  >
    <template #option="{ option }">
      <span class="min-w-0 flex-1 truncate">{{ option.label }}</span>
      <span
        v-if="option.hint"
        class="text-default-400 shrink-0 text-xs tabular-nums"
      >
        {{ option.hint }}
      </span>
    </template>
  </KunSelect>
</template>
