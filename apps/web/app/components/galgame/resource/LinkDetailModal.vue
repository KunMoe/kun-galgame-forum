<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_USER_TEXT_CHIP_CLASS
} from '~/constants/galgame'

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

const nuxtApp = useNuxtApp()

const { id: currentUserId } = usePersistUserStore()
const canEditAnyResource = useCan('resource.edit_any')
const canDeleteAnyResource = useCan('resource.delete_any')

const isEditOpen = ref(false)

const detail = ref<null | GalgameResourceDetailLink>(null)
const isFetching = ref(false)
const isExpired = computed(() => props.resource.status === 1)
const isOwner = computed(() => currentUserId === props.resource.user.id)
const canEdit = computed(() => isOwner.value || canEditAnyResource.value)
const canDelete = computed(() => isOwner.value || canDeleteAnyResource.value)

const providerName = computed(() => {
  const names = props.resource.provider_names
  return names && names.length > 0
    ? names.join(' / ')
    : props.resource.link_domain
})

const fetchDetail = async () => {
  if (detail.value || isFetching.value) return detail.value
  isFetching.value = true
  const result = await kunFetch<GalgameResourceDetailLink>(
    `/galgame-resource/${props.resource.id}/detail`,
    {
      method: 'GET',
      query: { galgame_resource_id: props.resource.id }
    }
  )
  isFetching.value = false
  if (result) detail.value = result
  return detail.value
}

defineExpose({ prefetch: fetchDetail })

const { status: reportStatus, report: reportExpire } =
  useReportResourceExpired()
const handleReportExpire = () =>
  reportExpire(props.resource.galgame_id, props.resource.id, () =>
    props.refresh()
  )

const handleDelete = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定删除 Galgame 资源链接吗？',
    '这将扣除发布者获得的 5 萌萌点, 并扣除其它人对资源链接的点赞影响, 此操作不可撤销。'
  )
  if (!res) return

  isFetching.value = true
  const result = await nuxtApp.runWithContext(() =>
    kunFetch(`/galgame/${props.resource.galgame_id}/resource`, {
      method: 'DELETE',
      query: { galgame_resource_id: props.resource.id }
    })
  )
  isFetching.value = false

  if (result) {
    nuxtApp.runWithContext(() => {
      useMessage('删除资源成功', 'success')
      props.refresh()
      open.value = false
    })
  }
}

const handleEdit = () => {
  if (!detail.value) return
  isEditOpen.value = true
}

const handleEditDone = () => {
  detail.value = null
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
            <KunAvatar :user="resource.user" size="lg" />
            <div class="flex flex-col">
              <span class="font-medium">{{ resource.user.name }}</span>
              <span class="text-default-500 text-xs">
                发布于 <KunTime :time="resource.created" />
              </span>
            </div>
          </div>
          <KunChip variant="flat" color="default" size="sm">
            <KunIcon name="lucide:download" />
            {{ resource.download }} 次下载
          </KunChip>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <KunChip color="primary" variant="flat">
            <KunIcon :name="GALGAME_RESOURCE_TYPE_ICON_MAP[resource.type]" />
            {{ KUN_GALGAME_RESOURCE_TYPE_MAP[resource.type] }}
          </KunChip>
          <KunChip color="warning" variant="flat" :class-name="KUN_USER_TEXT_CHIP_CLASS">
            <KunIcon name="lucide:database" />
            {{ resource.size }}
          </KunChip>
          <KunChip color="success" variant="flat">
            <KunIcon
              :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[resource.platform]"
            />
            {{ KUN_GALGAME_RESOURCE_PLATFORM_MAP[resource.platform] }}
          </KunChip>
          <KunChip color="secondary" variant="flat">
            <KunIcon name="lucide:globe" />
            {{ KUN_GALGAME_RESOURCE_LANGUAGE_MAP[resource.language] }}
          </KunChip>
        </div>

        <KunInfo
          v-if="resource.note"
          color="info"
          variant="flat"
          title="发布者备注 — 请先阅读"
        >
          <div class="space-y-1.5">
            <div ref="noteRef" :style="noteStyle" class="overflow-hidden">
              <KunContent compact :content="renderKatex(resource.note_html)" />
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

        <template v-else-if="detail">
          <KunAdAIFYBanner />

          <KunInfo color="primary" variant="flat" title="下载链接">
            <div class="space-y-1.5">
              <div
                v-for="(kun, index) in detail.link"
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
            v-if="detail.code || detail.password"
            class="flex flex-wrap items-center gap-2"
          >
            <KunCopy
              v-if="detail.code"
              variant="solid"
              :color="isExpired ? 'warning' : 'success'"
              :name="`提取码 ${detail.code}`"
              :text="detail.code"
            />
            <KunCopy
              v-if="detail.password"
              variant="solid"
              :color="isExpired ? 'warning' : 'success'"
              :name="`解压码 ${detail.password}`"
              :text="detail.password"
            />
          </div>

          <KunInfo title="鲲的小请求">
            <p>
              在您下载这部 Galgame 并游玩之后, 可否请您在本网站的
              <KunLink size="sm" :to="`/galgame/${resource.galgame_id}`">
                Galgame 评分页面
              </KunLink>
              为这部 Galgame 提交一个评分, 这将有助于我们把优秀的 Galgame
              推荐给更多人, 谢谢您的支持
            </p>
          </KunInfo>

          <GalgameResourceBuyLegitNotice
            :galgame-id="resource.galgame_id"
            :purchase-url="resource.dlsite_purchase_url"
            :coupon-url="resource.dlsite_coupon_url"
            :campaign-name="resource.dlsite_campaign_name"
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
      v-if="detail"
      v-model="isEditOpen"
      :galgame-id="resource.galgame_id"
      :resource="detail"
      :refresh="handleEditDone"
    />
  </KunModal>
</template>
