<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { WorkRef } from '#shared/utils/api/schemas'

type AutoOption = WorkRef & { value: string; label: string }

const props = defineProps<{
  excludeIds?: string[]
  placeholder?: string
}>()
const emits = defineEmits<{ select: [hit: WorkRef] }>()

const query = ref('')
const options = ref<AutoOption[]>([])
const isLoading = ref(false)
const api = useApiClient()
const workName = useWorkName()
const { allowsNsfw } = useContentStance()

let searchSeq = 0

const onSearch = async (raw: string) => {
  const kw = raw.trim().slice(0, 100)
  const seq = ++searchSeq
  if (!kw) {
    options.value = []
    isLoading.value = false
    return
  }
  isLoading.value = true
  const res = await settle(
    api.GET('/work-suggestions', {
      params: { query: { q: kw, include_nsfw: allowsNsfw.value } }
    })
  )
  if (seq !== searchSeq) return
  if (!res.ok) {
    reportProblem(res.problem)
    options.value = []
    isLoading.value = false
    return
  }
  const exclude = new Set((props.excludeIds ?? []).map(String))
  options.value = res.data.items
    .filter((o) => !exclude.has(o.id))
    .map((o) => ({
      ...o,
      value: o.id,
      label: workName(o)
    }))
  isLoading.value = false
}

const onSelect = (opt: AutoOption) => {
  emits('select', {
    id: opt.id,
    object: opt.object,
    display_name: opt.display_name,
    latin: opt.latin,
    localized: opt.localized,
    cover: opt.cover,
    is_nsfw: opt.is_nsfw
  })
  query.value = ''
  options.value = []
}
</script>

<template>
  <KunAutocomplete
    v-model="query"
    :options="options"
    :loading="isLoading"
    :debounce="300"
    manual-filter
    clearable
    :placeholder="placeholder ?? '输入游戏名搜索'"
    loading-text="搜索中…"
    no-result-text="无匹配结果"
    @search="onSearch"
    @select="onSelect"
  >
    <template #option="{ option }">
      <div class="flex items-center gap-3">
        <div class="bg-default-100 h-12 w-9 shrink-0 overflow-hidden rounded">
          <KunImage
            v-if="option.cover"
            :src="option.cover.url"
            :thumbhash="option.cover.thumbhash ?? undefined"
            :width="option.cover.width ?? 36"
            :height="option.cover.height ?? 48"
            object-fit="cover"
            class-name="h-full w-full"
          />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium">{{ option.label }}</p>
          <p v-if="option.is_nsfw" class="text-danger text-xs">NSFW</p>
        </div>
      </div>
    </template>
  </KunAutocomplete>
</template>
