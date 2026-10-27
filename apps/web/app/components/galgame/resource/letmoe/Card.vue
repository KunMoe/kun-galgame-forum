<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP,
  LETMOE_PROVIDER_LABEL_MAP,
  LETMOE_TYPE_LABEL_MAP,
  letmoeFeedbackUrl
} from '~/constants/galgameResource'
import type { LetmoeResource } from '#shared/utils/api/schemas'
import { patchNoteText } from '~/components/galgame/patch/noteText'
import { toKunUser } from '~/utils/userRef'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceRuntimeLabel,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const props = defineProps<{
  resource: LetmoeResource
  mode: 'direct' | 'netdisk'
}>()

const publisher = computed(() => toKunUser(props.resource.publisher))
const typeLabel = computed(
  () =>
    LETMOE_TYPE_LABEL_MAP[props.resource.letmoe_type] ??
    resourceTypeLabel(props.resource.letmoe_type)
)
const providerText = computed(() =>
  props.resource.providers
    .filter((p) => (props.mode === 'direct' ? p === 'direct' : p !== 'direct'))
    .map((p) => LETMOE_PROVIDER_LABEL_MAP[p] ?? p)
    .join(' / ')
)
const noteText = computed(() =>
  props.resource.note_markdown
    ? patchNoteText(props.resource.note_markdown).trim()
    : ''
)
const isNoteExpanded = ref(false)
const isNoteLong = computed(
  () => noteText.value.length > 120 || noteText.value.includes('\n')
)

const isDownloadOpen = ref(false)
const isCounted = ref(false)
const downloadCount = computed(
  () => props.resource.download_count + Number(isCounted.value)
)
</script>

<template>
  <KunCard
    :color="mode === 'direct' ? 'primary' : 'success'"
    content-class="space-y-3"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <UserHoverCard :user-id="publisher.id">
          <KunAvatar :user="publisher" size="md" />
        </UserHoverCard>
        <div class="flex flex-col leading-tight">
          <span class="text-sm font-medium">{{ publisher.name }}</span>
          <span class="text-default-500 text-xs">
            更新于 <KunTime :time="resource.updated_at" />
          </span>
        </div>
      </div>

      <KunTooltip position="left" text="该资源由 LetMoe·一起萌 提供, 点击前往">
        <KunButton
          :color="mode === 'direct' ? 'primary' : 'success'"
          variant="flat"
          size="sm"
          :href="resource.web_url"
          target="_blank"
          rel="noopener noreferrer"
        >
          <KunIcon :name="mode === 'direct' ? 'lucide:zap' : 'lucide:cloud'" />
          {{ mode === 'direct' ? '直链 · LetMoe·一起萌' : 'LetMoe·一起萌' }}
        </KunButton>
      </KunTooltip>
    </div>

    <p class="font-medium break-words">{{ resource.title }}</p>

    <div class="flex flex-wrap items-center gap-1.5">
      <KunChip color="primary" variant="flat">
        <KunIcon
          :name="
            GALGAME_RESOURCE_TYPE_ICON_MAP[resource.letmoe_type] ?? 'lucide:box'
          "
        />
        {{ typeLabel }}
      </KunChip>
      <KunChip v-if="resource.version_text" color="primary" variant="flat">
        {{ resource.version_text }}
      </KunChip>
      <KunChip v-if="resource.size" color="warning" variant="flat">
        <KunIcon name="lucide:database" />
        {{ resource.size }}
      </KunChip>
      <KunChip
        v-for="p in resource.platforms"
        :key="'p-' + p"
        color="success"
        variant="flat"
      >
        <KunIcon
          :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[p] ?? 'lucide:monitor'"
        />
        {{ resourcePlatformLabel(p) }}
      </KunChip>
      <KunChip
        v-for="lang in resource.languages"
        :key="'l-' + lang"
        color="secondary"
        variant="flat"
      >
        <KunIcon name="lucide:globe" />
        {{ resourceLanguageLabel(lang) }}
      </KunChip>
      <KunChip v-for="rt in resource.runtimes" :key="'r-' + rt" variant="flat">
        {{ resourceRuntimeLabel(rt) }}
      </KunChip>
      <KunChip
        v-if="mode === 'direct' && resource.composite_count"
        color="danger"
        variant="flat"
      >
        <KunIcon name="lucide:puzzle" />
        另有 {{ resource.composite_count }} 个补丁合成版, 点击直链下载选择
      </KunChip>
    </div>

    <KunInfo
      v-if="resource.content_warning"
      color="warning"
      variant="flat"
      title="发布者提示"
      :description="resource.content_warning"
    />

    <div v-if="noteText" class="space-y-1.5">
      <p
        :class="
          cn(
            'text-default-700 bg-default-100/60 rounded-md px-3 py-2 text-sm break-words whitespace-pre-line',
            !isNoteExpanded && 'line-clamp-3'
          )
        "
      >
        {{ noteText }}
      </p>
      <button
        v-if="isNoteLong"
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
        <span>{{ providerText }}</span>
      </div>

      <div class="flex flex-wrap items-center justify-end gap-1">
        <KunTooltip text="LetMoe 上的下载数">
          <div class="text-default-500 flex items-center gap-1 px-2 text-sm">
            <KunIcon name="lucide:download" />
            <span>{{ downloadCount }}</span>
          </div>
        </KunTooltip>

        <KunButton
          size="sm"
          variant="light"
          color="default"
          target="_blank"
          :href="letmoeFeedbackUrl(resource.web_url)"
        >
          <KunIcon name="lucide:message-square-warning" />
          到 LetMoe 反馈
        </KunButton>

        <KunButton
          size="sm"
          :color="mode === 'direct' ? 'primary' : 'success'"
          variant="solid"
          @click="isDownloadOpen = true"
        >
          <KunIcon
            :name="mode === 'direct' ? 'lucide:zap' : 'lucide:download-cloud'"
          />
          {{ mode === 'direct' ? '直链下载' : '获取资源' }}
        </KunButton>
      </div>
    </div>

    <GalgameResourceLetmoeDownloadModal
      v-model="isDownloadOpen"
      :resource="resource"
      :mode="mode"
      :download-count="downloadCount"
      @counted="isCounted = true"
    />
  </KunCard>
</template>
