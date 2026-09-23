<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { Work } from '#shared/utils/api/schemas'
import { getGalgameOriginalLanguageName } from '~/constants/galgame'
import { pickCatalogIntro } from '#shared/utils/catalogName'

const props = defineProps<{
  workId: number
  claimState?: string
  declineReason?: string
}>()

const isOpen = defineModel<boolean>({ required: true })
const api = useApiClient()
const namesOf = useCatalogName()

const detail = ref<Work | null>(null)
const isLoading = ref(false)
const loadedWorkId = ref(0)

const load = async () => {
  if (isLoading.value || loadedWorkId.value === props.workId) {
    return
  }
  isLoading.value = true
  const result = await settle(
    api.GET('/works/{work_id}', {
      params: {
        path: { work_id: String(props.workId) },
        query: { include_nsfw: true }
      }
    })
  )
  isLoading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  detail.value = result.data
  loadedWorkId.value = props.workId
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

const names = computed(() =>
  detail.value ? namesOf(detail.value) : { name: '', original: '' }
)

const intro = computed(
  () => pickCatalogIntro(detail.value?.intros ?? []) ?? detail.value?.intros[0]
)

const metaRows = computed(() => {
  const d = detail.value
  if (!d) {
    return []
  }
  return [
    {
      label: '发售日期',
      value: d.release_date || '待定'
    },
    {
      label: '原始语言',
      value: d.original_language
        ? getGalgameOriginalLanguageName(d.original_language)
        : ''
    },
    {
      label: '分级',
      value: d.content_rating === 'r18' ? 'R18' : '全年龄'
    },
    { label: '别名', value: d.aliases.join('、') }
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
          :src="detail.banner?.url || detail.cover?.url || ''"
          :alt="names.name"
          placeholder="/placeholder.webp"
          :thumbhash="detail.banner?.thumbhash ?? detail.cover?.thumbhash ?? undefined"
          class="w-full rounded-lg object-cover"
          :style="{ aspectRatio: '16/9' }"
        />

        <div class="space-y-1">
          <h4 class="text-lg font-medium">{{ names.name }}</h4>
          <p v-if="names.original" class="text-default-500 text-sm">
            {{ names.original }}
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
          <p class="text-default-700 whitespace-pre-line">
            {{ markdownToText(intro?.value ?? '', { preserveNewlines: true }) }}
          </p>
        </KunScrollShadow>
      </div>
    </div>
  </KunModal>
</template>
