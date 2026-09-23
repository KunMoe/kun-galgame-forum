<script setup lang="ts">
import {
  KUN_GALGAME_TOOLSET_TYPE_MAP,
  KUN_GALGAME_TOOLSET_LANGUAGE_MAP,
  KUN_GALGAME_TOOLSET_PLATFORM_MAP,
  KUN_GALGAME_TOOLSET_VERSION_MAP
} from '~/constants/toolset'
import { toolsetUpdateForm } from './rewriteStore'
import { settle } from '#shared/utils/api/problem'
import type {
  Toolset,
  ToolsetResourceSummary
} from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  id: string
  toolset: Toolset
}>()

const { id: userId } = usePersistUserStore()
const api = useApiClient()

const data = computed(() => props.toolset)
const canEditToolset = computed(() => data.value.viewer?.can_edit ?? false)
const canDeleteToolset = computed(() => data.value.viewer?.can_delete ?? false)

const resources = ref<ToolsetResourceSummary[]>([
  ...props.toolset.toolset_resources
])
watch(
  () => props.toolset.toolset_resources,
  (next) => {
    resources.value = [...next]
  }
)

const practicalityAverage = ref(props.toolset.practicality_average)
const practicalityDistribution = ref([
  ...props.toolset.practicality_distribution
])
const practicalityRating = ref(props.toolset.viewer?.practicality_rating ?? null)

watch(
  () => props.toolset,
  (next) => {
    practicalityAverage.value = next.practicality_average
    practicalityDistribution.value = [...next.practicality_distribution]
    practicalityRating.value = next.viewer?.practicality_rating ?? null
  }
)

const isDeleting = ref(false)
const handleDeleteToolset = async () => {
  if (!userId) {
    useAuthModal().open()
    return
  }

  const res = await useComponentMessageStore().alert(
    '确定删除该工具？',
    '删除这个工具将会消耗 3 萌萌点, 删除是永久性的, 不可撤销'
  )
  if (!res) {
    return
  }

  isDeleting.value = true
  const result = await settle(
    api.DELETE('/toolsets/{toolset_id}', {
      params: { path: { toolset_id: data.value.id } }
    })
  )
  isDeleting.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('已删除该工具', 'success')
  navigateTo('/toolset')
}

const handleRewriteToolset = () => {
  toolsetUpdateForm.toolset_id = data.value.id
  navigateTo('/edit/toolset/rewrite')
}

const showResourceModal = ref(false)
const handlePublishResource = () => {
  if (!userId) {
    useAuthModal().open()
    return
  }
  showResourceModal.value = true
}

const isSubmittingRate = ref(false)

const handleSetStar = async (val: number) => {
  if (!userId) {
    useAuthModal().open()
    return
  }
  if (isSubmittingRate.value) {
    return
  }
  isSubmittingRate.value = true
  const result = await settle(
    api.PUT('/toolsets/{toolset_id}/practicality', {
      params: { path: { toolset_id: props.id } },
      body: { rating: val }
    })
  )
  isSubmittingRate.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  practicalityAverage.value = result.data.practicality_average
  practicalityDistribution.value = [...result.data.practicality_distribution]
  practicalityRating.value = result.data.viewer?.practicality_rating ?? val
  useMessage(`已评分: ${val} 星`, 'success')
}

const handleResourceAdded = (res: ToolsetResourceSummary) => {
  resources.value = [res, ...resources.value]
}

const handleResourceDeleted = (toolsetResourceId: string) => {
  resources.value = resources.value.filter((r) => r.id !== toolsetResourceId)
}

const handleResourceUpdated = (res: ToolsetResourceSummary) => {
  const idx = resources.value.findIndex((r) => r.id === res.id)
  if (idx !== -1) {
    resources.value[idx] = res
  }
}
</script>

<template>
  <div class="space-y-4">
    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-6"
    >
      <div class="space-y-3">
        <h1 class="text-2xl leading-tight font-bold">{{ data.title }}</h1>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <KunChip color="secondary" size="sm">
            {{ KUN_GALGAME_TOOLSET_VERSION_MAP[data.release_channel] }}
          </KunChip>
          <KunChip color="success" size="sm">
            {{ KUN_GALGAME_TOOLSET_PLATFORM_MAP[data.platform] }}
          </KunChip>
          <KunChip color="primary" size="sm">
            {{ KUN_GALGAME_TOOLSET_LANGUAGE_MAP[data.interface_language] }}
          </KunChip>
          <KunChip color="danger" size="sm">
            {{ KUN_GALGAME_TOOLSET_TYPE_MAP[data.toolset_type] }}
          </KunChip>
        </div>

        <KunDivider class-name="my-6" />

        <ContentDocument :document="data.content" />
      </div>

      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div class="min-w-0 space-y-3 md:col-span-1">
          <ToolsetPracticalityChart
            :id="data.id"
            :distribution="practicalityDistribution"
            :average="practicalityAverage"
            :rating="practicalityRating"
          />
        </div>

        <div class="min-w-0 space-y-6 md:col-span-1">
          <div class="space-y-2">
            <h3 class="font-semibold">发布者</h3>
            <KunUserChip :user="toKunUser(data.author)" />
          </div>

          <div v-if="data.homepage_urls.length" class="space-y-2">
            <h3 class="font-semibold">主页 / 项目</h3>
            <div class="flex flex-col gap-2">
              <KunLink
                v-for="(url, idx) in data.homepage_urls"
                :key="idx"
                :to="url"
                target="_blank"
                underline="hover"
              >
                {{ url }}
              </KunLink>
            </div>
          </div>

          <div v-if="data.aliases.length" class="space-y-2">
            <h3 class="font-semibold">别名</h3>
            <div class="flex flex-wrap items-center gap-2">
              <KunChip
                v-for="(a, i) in data.aliases"
                :key="i"
                color="default"
                size="sm"
              >
                {{ a }}
              </KunChip>
            </div>
          </div>

          <div class="space-y-2">
            <h3 class="font-semibold">实用性评分</h3>
            <p class="text-default-500 text-sm">点击以评分</p>
            <div class="flex items-center gap-2">
              <KunRating
                :model-value="practicalityRating ?? undefined"
                @set="handleSetStar"
              />
              <KunChip size="sm" variant="flat">
                {{ practicalityRating ?? '—' }} / 5
              </KunChip>
            </div>
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="text-default-500 text-sm">
          {{ `${formatNumber(data.view_count)} 浏览` }}
          ·
          {{ `${formatNumber(data.download_count)} 下载` }}
        </div>
        <div class="flex gap-1">
          <KunButton @click="handlePublishResource">上传 / 添加资源</KunButton>
          <KunButton
            v-if="canEditToolset"
            variant="flat"
            @click="handleRewriteToolset"
          >
            修改
          </KunButton>
          <KunButton
            v-if="canDeleteToolset"
            color="danger"
            variant="flat"
            :loading="isDeleting"
            @click="handleDeleteToolset"
          >
            删除
          </KunButton>
        </div>
      </div>
    </KunCard>

    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <ToolsetResourceList
        :toolset-id="data.id"
        :resources="resources"
        @deleted="handleResourceDeleted"
        @updated="handleResourceUpdated"
      />
    </KunCard>

    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <ToolsetCommentCommunityContainer :toolset-id="Number(data.id)" />
    </KunCard>

    <KunModal
      :model-value="showResourceModal"
      @update:model-value="(v) => (showResourceModal = v)"
      inner-class-name="max-w-2xl"
      :is-dismissable="false"
    >
      <ToolsetResourceContainer
        :toolset-id="data.id"
        @on-close="() => (showResourceModal = false)"
        @on-success="handleResourceAdded"
      />
    </KunModal>
  </div>
</template>
