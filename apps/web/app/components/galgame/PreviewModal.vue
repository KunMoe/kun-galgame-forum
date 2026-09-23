<script setup lang="ts">
const props = defineProps<{
  workId: number
  claimState?: string
  declineReason?: string
}>()

const isOpen = defineModel<boolean>({ required: true })

const detail = ref<GalgameDetail | null>(null)
const isLoading = ref(false)
const loadedWorkId = ref(0)

const load = async () => {
  if (isLoading.value || loadedWorkId.value === props.workId) {
    return
  }
  isLoading.value = true
  detail.value = await kunFetch<GalgameDetail>(`/galgame/${props.workId}`)
  isLoading.value = false
  if (detail.value) {
    loadedWorkId.value = props.workId
  }
}

watch(
  [isOpen, () => props.workId],
  ([open]) => {
    if (open) {
      load()
    }
  },
  { immediate: true }
)

const badge = computed(() => galgameClaimStateBadge(props.claimState))

const title = computed(() => detail.value?.name ?? '')

const originalName = computed(() => detail.value?.name_original ?? '')

const metaRows = computed(() => {
  const d = detail.value
  if (!d) {
    return []
  }
  return [
    {
      label: '发售日期',
      value: d.release_date || (d.release_date_tba ? '待定' : '')
    },
    { label: '原始语言', value: d.original_language },
    { label: '分级', value: d.age_limit === 'r18' ? 'R18' : '全年龄' },
    { label: '别名', value: d.alias.join('、') }
  ].filter((row) => row.value)
})
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="w-full max-w-3xl">
    <div class="space-y-4">
      <div class="flex flex-wrap items-center gap-2">
        <h3 class="text-xl font-medium">审核预览</h3>
        <KunChip size="xs" variant="flat" :color="badge.color">
          {{ badge.label }}
        </KunChip>
        <span class="text-default-500 text-sm">galgame_id: {{ workId }}</span>
      </div>

      <KunLoading v-if="isLoading" description="加载中…" />

      <KunInfo
        v-else-if="!detail"
        color="danger"
        title="无法加载预览"
        description="这个条目可能已被撤回或删除, 也可能是 Galgame 资料库暂时不可用。"
      />

      <div v-else class="space-y-4">
        <KunImage
          :src="getEffectiveBanner(detail)"
          :alt="title"
          placeholder="/placeholder.webp"
          :thumbhash="detail.effective_banner_thumbhash"
          class="w-full rounded-lg object-cover"
          :style="{ aspectRatio: '16/9' }"
        />

        <div class="space-y-1">
          <h4 class="text-lg font-medium">{{ title }}</h4>
          <p v-if="originalName" class="text-default-500 text-sm">
            {{ originalName }}
          </p>
        </div>

        <div v-if="metaRows.length" class="grid gap-2 sm:grid-cols-2">
          <div
            v-for="row in metaRows"
            :key="row.label"
            class="bg-default-100 rounded-lg px-3 py-2"
          >
            <div class="text-default-500 text-xs">{{ row.label }}</div>
            <div class="text-sm break-all">{{ row.value }}</div>
          </div>
        </div>

        <div
          v-if="declineReason"
          class="text-danger bg-danger/10 rounded-md px-3 py-2 text-sm"
        >
          被拒原因: {{ declineReason }}
        </div>

        <KunScrollShadow class="max-h-80">
          <KunContent :content="detail.introduction[0]?.intro ?? ''" />
        </KunScrollShadow>
      </div>
    </div>
  </KunModal>
</template>
