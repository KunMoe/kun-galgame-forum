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

const isFetching = ref(false)
const { id } = usePersistUserStore()

const nuxtApp = useNuxtApp()

const isExpired = computed(() => props.resource.status === 1)
const isOwner = computed(() => id === props.resource.user.id)

const providerName = computed(() => {
  const names = props.resource.provider_names
  return names && names.length > 0
    ? names.join(' / ')
    : props.resource.link_domain
})

const NOTE_COLLAPSED_MAX_HEIGHT = 100
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

onMounted(() => {
  if (!noteRef.value) return
  noteResizeObserver = new ResizeObserver(() => measureNoteOverflow())
  noteResizeObserver.observe(noteRef.value)
  measureNoteOverflow()
})

onBeforeUnmount(() => {
  noteResizeObserver?.disconnect()
  noteResizeObserver = null
})

watch(
  () => props.resource.note,
  () => {
    isNoteExpanded.value = false
    nextTick(measureNoteOverflow)
  }
)

const isDetailOpen = ref(false)
const isOpeningDetail = ref(false)
const detailModalRef = ref<{ prefetch: () => Promise<unknown> } | null>(null)

const openDetail = async () => {
  if (isOpeningDetail.value) return
  isOpeningDetail.value = true
  try {
    await nuxtApp.runWithContext(() => detailModalRef.value?.prefetch())
    isDetailOpen.value = true
  } finally {
    isOpeningDetail.value = false
  }
}

const handleMarkValid = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定重新标记资源链接有效吗？',
    '若您修复了资源链接，您可以重新标记资源链接有效。'
  )
  if (!res) return

  isFetching.value = true
  const result = await nuxtApp.runWithContext(() =>
    kunFetch(`/galgame/${props.resource.galgame_id}/resource/valid`, {
      method: 'PUT',
      body: { galgame_resource_id: props.resource.id }
    })
  )
  isFetching.value = false

  if (result) {
    nuxtApp.runWithContext(() => {
      useMessage(10548, 'success')
      props.refresh()
    })
  }
}
</script>

<template>
  <KunCard :color="isExpired ? 'warning' : 'success'" content-class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <KunAvatar :user="resource.user" size="md" />
        <div class="flex flex-col leading-tight">
          <span class="text-sm font-medium">{{ resource.user.name }}</span>
          <span class="text-default-500 text-xs">
            <KunTime :time="resource.created" />
          </span>
        </div>
      </div>

      <KunTooltip
        position="left"
        :text="isExpired ? '该资源已被标记失效' : '该资源链接有效'"
      >
        <KunChip
          :color="isExpired ? 'warning' : 'success'"
          variant="flat"
          size="sm"
        >
          <KunIcon
            :name="isExpired ? 'lucide:triangle-alert' : 'lucide:circle-check'"
          />
          {{ isExpired ? '失效' : '有效' }}
        </KunChip>
      </KunTooltip>
    </div>

    <div class="flex flex-wrap items-center gap-1.5">
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

    <div v-if="resource.note" class="space-y-1.5">
      <p
        ref="noteRef"
        :style="noteStyle"
        class="text-default-700 bg-default-100/60 overflow-hidden rounded-md px-3 py-2 text-sm break-words whitespace-pre-line"
      >
        {{ markdownToText(resource.note, { preserveNewlines: true }) }}
      </p>

      <button
        v-if="isNoteOverflowing"
        type="button"
        class="text-default-500 hover:text-primary flex items-center gap-1 px-1 text-xs transition-colors"
        @click="isNoteExpanded = !isNoteExpanded"
      >
        <KunIcon
          :name="isNoteExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
        />
        {{ isNoteExpanded ? '收起' : '展开全部' }}
      </button>
    </div>

    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="text-default-500 flex items-center gap-1.5 text-sm">
        <KunIcon name="lucide:hard-drive" />
        <span>{{ providerName }}</span>
      </div>

      <div class="flex items-center gap-1">
        <KunTooltip text="资源下载数">
          <div class="text-default-500 flex items-center gap-1 px-2 text-sm">
            <KunIcon name="lucide:download" />
            <span>{{ resource.download }}</span>
          </div>
        </KunTooltip>

        <GalgameResourceLike
          v-if="!isOwner"
          :galgame-id="resource.galgame_id"
          :galgame-resource-id="resource.id"
          :target-user-id="resource.user.id"
          :is-liked="resource.is_liked"
          :like-count="resource.like_count"
        />

        <KunButton
          v-if="isOwner && isExpired"
          size="sm"
          variant="flat"
          color="success"
          :loading="isFetching"
          @click="handleMarkValid"
        >
          重新标记有效
        </KunButton>

        <KunButton
          size="sm"
          :color="isExpired ? 'warning' : 'primary'"
          variant="solid"
          :loading="isOpeningDetail"
          @click="openDetail"
        >
          <KunIcon name="lucide:download-cloud" />
          获取资源
        </KunButton>
      </div>
    </div>

    <GalgameResourceLinkDetailModal
      ref="detailModalRef"
      v-model="isDetailOpen"
      :resource="resource"
      :refresh="refresh"
    />
  </KunCard>
</template>
