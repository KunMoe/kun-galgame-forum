<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import { KUN_USER_TEXT_CHIP_CLASS } from '~/constants/galgame'
import { settle } from '#shared/utils/api/problem'
import type {
  GalgameResource,
  GalgameResourceDownload
} from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser } from '~/utils/userRef'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const props = defineProps<{
  resource: GalgameResource
  refresh: () => void
}>()

const open = defineModel<boolean>({ required: true })

const NOTE_COLLAPSED_MAX_HEIGHT = 240
const noteRef = ref<HTMLElement | null>(null)
const isNoteExpanded = ref(false)
const isNoteOverflowing = ref(false)
let noteResizeObserver: ResizeObserver | null = null

const measureNoteOverflow = () => {
  const el = noteRef.value
  if (!el) {
    isNoteOverflowing.value = false
    return
  }
  isNoteOverflowing.value = el.scrollHeight > NOTE_COLLAPSED_MAX_HEIGHT
}

const noteStyle = computed(() => {
  if (!isNoteOverflowing.value || isNoteExpanded.value) return undefined
  return {
    maxHeight: `${NOTE_COLLAPSED_MAX_HEIGHT}px`,
    overflow: 'hidden'
  }
})

const teardownNoteObserver = () => {
  noteResizeObserver?.disconnect()
  noteResizeObserver = null
}

watch(open, (isOpen) => {
  if (!isOpen) {
    teardownNoteObserver()
    return
  }
  isNoteExpanded.value = false
  nextTick(() => {
    if (!noteRef.value) return
    teardownNoteObserver()
    noteResizeObserver = new ResizeObserver(() => measureNoteOverflow())
    noteResizeObserver.observe(noteRef.value)
    measureNoteOverflow()
  })
})

onBeforeUnmount(teardownNoteObserver)

const { id: currentUserId } = usePersistUserStore()
const api = useApiClient()

const isEditOpen = ref(false)

const download = ref<GalgameResourceDownload | null>(null)
const downloadsHere = ref(0)
const isFetching = ref(false)
const isExpired = computed(() => props.resource.state === 'expired')
const isOwner = computed(
  () => currentUserId === Number(props.resource.author.id)
)
const canEdit = computed(() => props.resource.viewer?.can_edit ?? false)
const canDelete = computed(() => props.resource.viewer?.can_delete ?? false)
const author = computed(() => toKunUser(props.resource.author))
const workId = computed(() => props.resource.work?.id ?? '')
const hasNote = computed(
  () => contentPlainText(props.resource.content).trim().length > 0
)

const providerName = computed(() => {
  const names = props.resource.provider_names
  return names.length > 0 ? names.join(' / ') : ''
})

const fetchDownload = async () => {
  if (download.value || isFetching.value) return download.value
  isFetching.value = true
  const result = await settle(
    api.POST('/galgame-resources/{resource_id}/downloads', {
      params: { path: { resource_id: props.resource.id } }
    })
  )
  isFetching.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return null
  }
  download.value = result.data
  downloadsHere.value += 1
  return download.value
}

defineExpose({ prefetch: fetchDownload })

const { status: reportStatus, report: reportExpire } =
  useReportResourceExpired()
const handleReportExpire = () =>
  reportExpire(props.resource.id, () => props.refresh())

const handleDelete = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定删除 Galgame 资源链接吗？',
    '这将扣除发布者获得的 3 萌萌点, 此操作不可撤销。'
  )
  if (!res) return

  isFetching.value = true
  const result = await settle(
    api.DELETE('/galgame-resources/{resource_id}', {
      params: { path: { resource_id: props.resource.id } }
    })
  )
  isFetching.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除资源成功', 'success')
  props.refresh()
  open.value = false
}

const handleEdit = () => {
  isEditOpen.value = true
}

const handleEditDone = () => {
  download.value = null
  props.refresh()
  open.value = false
}
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-2xl w-[92vw] !p-0">
    <div class="flex flex-col">
      <div
        :class="
          cn(
            'flex items-center justify-between gap-3 px-5 py-3',
            isExpired
              ? 'bg-warning/10 text-warning-700 dark:text-warning'
              : 'bg-success/10 text-success-700 dark:text-success'
          )
        "
      >
        <div class="flex items-center gap-2">
          <KunIcon
            :name="isExpired ? 'lucide:triangle-alert' : 'lucide:circle-check'"
            class="text-xl"
          />
          <span class="text-base font-medium">
            {{ isExpired ? '该资源链接已被标记失效' : '该资源链接可用' }}
          </span>
        </div>
        <KunChip
          v-if="providerName"
          variant="flat"
          :color="isExpired ? 'warning' : 'success'"
          size="sm"
        >
          {{ providerName }}
        </KunChip>
      </div>

      <div class="space-y-5 p-5">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-3">
            <KunAvatar :user="author" size="lg" />
            <div class="flex flex-col">
              <span class="font-medium">{{ author.name }}</span>
              <span class="text-default-500 text-xs">
                发布于 <KunTime :time="resource.created_at" />
              </span>
            </div>
          </div>
          <KunChip variant="flat" color="default" size="sm">
            <KunIcon name="lucide:download" />
            {{ resource.download_count + downloadsHere }} 次下载
          </KunChip>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <KunChip color="primary" variant="flat">
            <KunIcon
              :name="GALGAME_RESOURCE_TYPE_ICON_MAP[resource.resource_type]"
            />
            {{ resourceTypeLabel(resource.resource_type) }}
          </KunChip>
          <KunChip
            color="warning"
            variant="flat"
            :class-name="KUN_USER_TEXT_CHIP_CLASS"
          >
            <KunIcon name="lucide:database" />
            {{ resource.size }}
          </KunChip>
          <KunChip
            v-for="p in resource.resource_platforms"
            :key="'p-' + p"
            color="success"
            variant="flat"
          >
            <KunIcon :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[p]" />
            {{ resourcePlatformLabel(p) }}
          </KunChip>
          <KunChip
            v-for="lang in resource.resource_languages"
            :key="'l-' + lang"
            color="secondary"
            variant="flat"
          >
            <KunIcon name="lucide:globe" />
            {{ resourceLanguageLabel(lang) }}
          </KunChip>
        </div>

        <KunInfo
          v-if="hasNote"
          color="info"
          variant="flat"
          title="发布者备注 — 请先阅读"
        >
          <div class="space-y-1.5">
            <div ref="noteRef" :style="noteStyle" class="overflow-hidden">
              <ContentDocument compact :document="resource.content" />
            </div>

            <button
              v-if="isNoteOverflowing"
              type="button"
              class="text-default-500 hover:text-primary flex items-center gap-1 px-1 text-xs transition-colors"
              @click="isNoteExpanded = !isNoteExpanded"
            >
              <KunIcon
                :name="
                  isNoteExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'
                "
              />
              {{ isNoteExpanded ? '收起' : '展开全部' }}
            </button>
          </div>
        </KunInfo>

        <div v-if="isFetching" class="flex justify-center py-8">
          <KunLoading />
        </div>

        <template v-else-if="download">
          <KunAdAIFYBanner />

          <KunInfo color="primary" variant="flat" title="下载链接">
            <p
              v-if="!download.download_urls.length"
              class="text-default-500 text-sm"
            >
              这个资源没有登记下载链接，链接可能写在提取码或备注里
            </p>
            <div v-else class="space-y-1.5">
              <div
                v-for="(kun, index) in download.download_urls"
                :key="index"
                class="flex items-start gap-2"
              >
                <KunIcon
                  name="lucide:external-link"
                  class="text-primary mt-1 shrink-0"
                />
                <KunLink
                  :to="kun"
                  target="_blank"
                  rel="noopener noreferrer"
                  size="sm"
                  class-name="break-all"
                >
                  {{ kun }}
                </KunLink>
              </div>
            </div>
          </KunInfo>

          <div
            v-if="download.extraction_code || download.archive_password"
            class="flex flex-wrap items-center gap-2"
          >
            <KunCopy
              v-if="download.extraction_code"
              variant="solid"
              :color="isExpired ? 'warning' : 'success'"
              :name="`提取码 ${download.extraction_code}`"
              :text="download.extraction_code"
            />
            <KunCopy
              v-if="download.archive_password"
              variant="solid"
              :color="isExpired ? 'warning' : 'success'"
              :name="`解压码 ${download.archive_password}`"
              :text="download.archive_password"
            />
          </div>

          <KunInfo title="鲲的小请求">
            <p>
              在您下载这部 Galgame 并游玩之后, 可否请您在本网站的
              <KunLink size="sm" :to="`/galgame/${workId}`">
                Galgame 评分页面
              </KunLink>
              为这部 Galgame 提交一个评分, 这将有助于我们把优秀的 Galgame
              推荐给更多人, 谢谢您的支持
            </p>
          </KunInfo>

          <GalgameResourceBuyLegitNotice
            :work-id="workId"
            :purchase-url="resource.dlsite?.purchase_url"
            :coupon-url="resource.dlsite?.coupon_url ?? undefined"
          />
        </template>

        <div class="flex flex-wrap items-center justify-between gap-1">
          <div class="flex flex-wrap items-center gap-1">
            <KunButton
              v-if="canEdit"
              variant="light"
              color="default"
              @click="handleEdit"
            >
              <KunIcon name="lucide:pencil" />
              编辑
            </KunButton>
            <KunButton
              v-if="canDelete"
              variant="light"
              color="danger"
              :loading="isFetching"
              @click="handleDelete"
            >
              <KunIcon name="lucide:trash-2" />
              删除
            </KunButton>
            <KunButton
              v-if="!isOwner && !isExpired"
              variant="light"
              color="warning"
              :loading="reportStatus === 'checking'"
              :disabled="reportStatus === 'checking'"
              @click="handleReportExpire"
            >
              <KunIcon name="lucide:triangle-alert" />
              报告失效
            </KunButton>
          </div>

          <GalgameResourceExpireStatus :status="reportStatus" />

          <div class="flex flex-wrap items-center gap-1">
            <KunButton
              variant="light"
              color="default"
              :href="`/galgame/resource/${resource.id}`"
            >
              <KunIcon name="lucide:external-link" />
              查看详情页
            </KunButton>
            <KunButton variant="solid" color="default" @click="open = false">
              关闭
            </KunButton>
          </div>
        </div>
      </div>
    </div>

    <GalgameResourceLinkEditModal
      v-model="isEditOpen"
      :work-id="workId"
      :resource-id="resource.id"
      :refresh="handleEditDone"
    />
  </KunModal>
</template>
