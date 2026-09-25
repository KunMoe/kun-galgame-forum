<script setup lang="ts">
import type { KunSelectValue } from '@kungal/ui-vue'
import type { TraitSummary } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { traitContext } from '~/utils/galgame/trait'

const props = defineProps<{
  selectedIds: number[]
  /** Whatever the caller has already resolved; listed when idle. */
  selectedItems: TraitSummary[]
  max: number
}>()

const emit = defineEmits<{ toggle: [trait: TraitSummary] }>()

const query = ref('')
const found = ref<TraitSummary[]>([])
const loading = ref(false)
const failed = ref(false)
const api = useApiClient()
const { allowsNsfw } = useContentStance()

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
  const res = await settle(
    api.GET('/traits', {
      params: { query: { q, limit: 30, include_nsfw: allowsNsfw.value } }
    })
  )
  if (current !== latest) {
    return
  }
  failed.value = !res.ok
  found.value = res.ok ? res.data.items.filter((t) => t.is_searchable) : []
  loading.value = false
}

const full = computed(() => props.selectedIds.length >= props.max)

const known = computed(() => {
  const map = new Map<number, TraitSummary>()
  for (const t of [...props.selectedItems, ...found.value]) {
    map.set(Number(t.id), t)
  }
  return map
})

const options = computed(() => {
  const chosen = props.selectedIds
    .map((id) => known.value.get(id))
    .filter((t) => !!t)
  const rest = found.value.filter(
    (t) => !props.selectedIds.includes(Number(t.id))
  )
  return [...chosen, ...rest].map((t) => ({
    value: Number(t.id),
    label: catalogVocabularyName(t),
    context: traitContext(t),
    hint: t.character_count.toLocaleString('en-US'),
    disabled: full.value && !props.selectedIds.includes(Number(t.id))
  }))
})

const pick = (value: KunSelectValue) => {
  const trait = known.value.get(Number(value))
  if (trait) {
    emit('toggle', trait)
  }
}

// The URL owns the selection; see FilterEntityMenu.
const keepControlled = () => {}

const emptyText = computed(() => {
  if (!query.value.trim()) {
    return '输入关键词以搜索属性, 例如 银发 / 傲娇 / 眼镜'
  }
  if (failed.value) {
    return '属性搜索没能完成, 请稍后重试'
  }
  return allowsNsfw.value
    ? '没有找到匹配的属性'
    : '没有找到匹配的属性, 成人向属性需在设置面板开启 NSFW'
})
</script>

<template>
  <KunSelect
    :model-value="selectedIds"
    :options="options"
    multiple
    icon="lucide:sparkles"
    placeholder="添加属性"
    aria-label="按属性筛选角色"
    :full-width="false"
    :max-visible-tags="0"
    :class-names="{ trigger: filterPillClass(selectedIds.length > 0) }"
    :search-placeholder="
      full ? `最多同时筛选 ${max} 个属性` : '搜索属性, 例如 黑长直'
    "
    :no-result-text="emptyText"
    :loading="loading"
    :debounce="300"
    loading-text="正在检索属性…"
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
      <span class="min-w-0 flex-1">
        <span class="block truncate">{{ option.label }}</span>
        <span
          v-if="option.context"
          class="text-default-400 block truncate text-xs"
        >
          {{ option.context }}
        </span>
      </span>
      <span class="text-default-400 shrink-0 text-xs tabular-nums">
        {{ option.hint }}
      </span>
    </template>
  </KunSelect>
</template>
