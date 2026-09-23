<script setup lang="ts">
import type { ListMoyuPatch } from '#shared/utils/api/schemas'
import {
  SUPPORTED_TYPE_MAP,
  SUPPORTED_PLATFORM_MAP,
  SUPPORTED_LANGUAGE_MAP
} from './constant'
import { patchNoteText } from './noteText'

const props = defineProps<{
  workId: number
}>()

const emit = defineEmits<{
  'has-resource': [boolean]
  'update:loading': [boolean]
}>()

// A failed lookup is not reported: moyu is another site, and the tab is only
// drawn when there is something in it.
const { data, status } = useApi<ListMoyuPatch>(
  () => `galgame-moyu-patches:${props.workId}`,
  (api, { signal }) =>
    api.GET('/works/{work_id}/moyu-patches', {
      params: { path: { work_id: String(props.workId) } },
      signal
    }),
  { lazy: true, server: false }
)

const pageUrl = computed(() => data.value?.items[0]?.web_url)
const resources = computed(
  () => data.value?.items.flatMap((patch) => patch.resources) ?? []
)

watchEffect(() => emit('update:loading', status.value === 'pending'))
watch(resources, (list) => emit('has-resource', list.length > 0), {
  immediate: true
})

const expandedNotes = ref<Set<string>>(new Set())
const isNoteExpanded = (id: string) => expandedNotes.value.has(id)
const toggleNote = (id: string) => {
  if (expandedNotes.value.has(id)) {
    expandedNotes.value.delete(id)
  } else {
    expandedNotes.value.add(id)
  }
}
const isNoteLong = (note: string) => note.length > 80 || note.includes('\n')

const STORAGE_MAP: Record<string, string> = {
  s3: 'S3 对象存储',
  user: '网盘下载'
}
</script>

<template>
  <div v-if="resources.length" class="flex flex-col gap-3">
    <KunHeader name="补丁资源下载" scale="h2">
      <template #endContent>
        <p class="text-default-500 text-sm">
          下面是从
          <KunLink size="sm" :href="pageUrl" target="_blank">
            鲲 Galgame 补丁
          </KunLink>
          获取到的 Galgame 补丁资源, 请杂鱼点击前往本站的补丁网站下载
        </p>
      </template>
    </KunHeader>

    <!-- One card per resource, matching this site's own resource list. The rows
         used to be bare blocks separated by a hairline KunDivider, and readers
         「时常会有点错行进错上行的资源列表」: with no bounded container the
         download button at the bottom of a row reads as belonging to the name
         below the rule rather than the one above it. -->
    <KunCard
      v-for="resource in resources"
      :key="resource.id"
      :is-transparent="false"
      :is-hoverable="false"
      content-class="space-y-2"
    >
      <p v-if="resource.name" class="font-medium break-words">
        {{ resource.name }}
      </p>
      <div class="flex flex-wrap items-center justify-between">
        <div class="flex flex-wrap items-center gap-1 rounded-lg">
          <KunChip
            v-for="t in resource.types"
            :key="t"
            size="sm"
            variant="flat"
            color="primary"
          >
            {{ SUPPORTED_TYPE_MAP[t] ?? t }}
          </KunChip>
          <KunChip size="sm" variant="flat" color="warning">
            <KunIcon name="lucide:database" />
            {{ resource.size }}
          </KunChip>
          <KunChip
            v-for="p in resource.platforms"
            :key="p"
            size="sm"
            variant="flat"
            color="success"
          >
            {{ SUPPORTED_PLATFORM_MAP[p] ?? p }}
          </KunChip>
          <KunChip
            v-for="l in resource.languages"
            :key="l"
            size="sm"
            variant="flat"
            color="secondary"
          >
            {{ SUPPORTED_LANGUAGE_MAP[l] ?? l }}
          </KunChip>
          <KunChip
            v-if="resource.model_name"
            size="sm"
            variant="flat"
            color="danger"
          >
            <KunIcon name="lucide:bot" />
            {{ resource.model_name }}
          </KunChip>
          <KunChip size="sm" variant="flat" color="default">
            <KunIcon name="lucide:hard-drive" />
            {{ STORAGE_MAP[resource.storage] ?? resource.storage }}
          </KunChip>
        </div>

        <div class="ml-auto flex items-center gap-1">
          <KunTooltip text="下载数">
            <KunButton
              variant="light"
              color="default"
              size="sm"
              class-name="gap-1"
            >
              <KunIcon name="lucide:download" />
              <span>{{ resource.download_count }}</span>
            </KunButton>
          </KunTooltip>
        </div>
      </div>

      <KunInfo
        v-if="resource.note_markdown"
        color="info"
        variant="flat"
        title="发布者备注 — 请先阅读"
      >
        <div class="space-y-1.5">
          <p
            class="text-sm break-words whitespace-pre-wrap"
            :class="{ 'line-clamp-3': !isNoteExpanded(resource.id) }"
          >
            {{ patchNoteText(resource.note_markdown) }}
          </p>
          <button
            v-if="isNoteLong(resource.note_markdown)"
            type="button"
            class="text-default-500 hover:text-primary flex items-center gap-1 px-1 text-xs transition-colors"
            @click="toggleNote(resource.id)"
          >
            <KunIcon
              :name="
                isNoteExpanded(resource.id)
                  ? 'lucide:chevron-up'
                  : 'lucide:chevron-down'
              "
            />
            {{ isNoteExpanded(resource.id) ? '收起' : '展开全部' }}
          </button>
        </div>
      </KunInfo>

      <div class="flex justify-between">
        <div class="flex gap-2">
          <KunAvatar :user="toKunUser(resource.publisher)" />

          <div class="flex flex-col">
            <span class="text-xs">
              {{ toKunUser(resource.publisher).name }}
            </span>
            <span class="text-default-500 text-xs">
              资源更新于
              <KunTime :time="resource.updated_at" type="datetime" show-year />
            </span>
          </div>
        </div>

        <KunButton
          size="sm"
          variant="flat"
          target="_blank"
          :href="resource.web_url"
          :icon="true"
        >
          前往下载页面
          <template #icon>
            <KunIcon name="lucide:external-link" />
          </template>
        </KunButton>
      </div>
    </KunCard>
  </div>
</template>
