<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { Section } from '#shared/utils/api/schemas'
import type { SectionForm } from './types'

const api = useApiClient()
const { sections, refresh } = useTopicSections()

const editing = ref<Section | null>(null)
const isEditOpen = ref(false)
const isSaving = ref(false)

const openEdit = (section: Section) => {
  editing.value = section
  isEditOpen.value = true
}

const patch = async (
  key: Section['section'],
  body: Partial<SectionForm> & { is_archived?: boolean }
) => {
  const result = await settle(
    api.PATCH('/admin/sections/{section}', {
      params: { path: { section: key } },
      body
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  await refresh()
  return true
}

const saveSection = async (body: SectionForm) => {
  if (!editing.value) {
    return
  }
  isSaving.value = true
  const ok = await patch(editing.value.section, body)
  isSaving.value = false
  if (ok) {
    isEditOpen.value = false
    useMessage('分区已更新', 'success')
  }
}

const toggleArchive = async (section: Section) => {
  const next = !section.is_archived
  const confirmed = await useComponentMessageStore().alert(
    next
      ? `确定归档分区「${section.label}」吗？`
      : `确定恢复分区「${section.label}」吗？`,
    next
      ? '归档后分区里的话题保留, 但不能再发布新话题到这个分区。'
      : '恢复后用户可以再次发布话题到这个分区。'
  )
  if (confirmed && (await patch(section.section, { is_archived: next }))) {
    useMessage(next ? '分区已归档' : '分区已恢复', 'success')
  }
}

const move = async (index: number, delta: number) => {
  const order = sections.value.map((s) => s.section)
  ;[order[index], order[index + delta]] = [order[index + delta]!, order[index]!]
  const result = await settle(
    api.PUT('/admin/section-order', { body: { sections: order } })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  await refresh()
}
</script>

<template>
  <div class="space-y-3">
    <div
      v-for="(section, index) in sections"
      :key="section.section"
      :class="
        cn(
          'border-default-200 bg-background flex items-start gap-3 rounded-xl border p-3',
          section.is_archived && 'opacity-60'
        )
      "
    >
      <img
        v-if="section.banner"
        :src="section.banner.url"
        alt=""
        class="hidden h-14 w-24 shrink-0 rounded-lg object-cover sm:block"
      />
      <KunIcon v-else :name="section.icon_name" class="mt-1 shrink-0" />
      <div class="min-w-0 flex-1 space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <KunIcon :name="section.icon_name" class="size-4" />
          <span class="font-medium">{{ section.label }}</span>
          <span class="text-default-400 font-mono text-xs">
            {{ section.section }}
          </span>
          <KunChip size="sm" :color="section.color">颜色</KunChip>
          <KunChip
            v-if="section.moemoepoint_on_create < 0"
            size="sm"
            color="warning"
          >
            消耗 {{ -section.moemoepoint_on_create }} 萌萌点
          </KunChip>
          <KunChip v-if="section.is_hidden_from_feeds" size="sm">
            不进入主页与全部话题
          </KunChip>
          <KunChip v-if="section.is_archived" size="sm">已归档</KunChip>
          <KunChip v-if="section.mascots.length" size="sm">
            站娘 {{ section.mascots.length }} 张
          </KunChip>
          <span class="text-default-500 text-xs">
            {{ section.topic_count }} 个话题
          </span>
        </div>
        <p class="text-default-500 line-clamp-2 text-xs">
          {{ section.description }}
        </p>
      </div>
      <div class="flex shrink-0 gap-1">
        <KunButton
          size="sm"
          variant="light"
          :is-icon-only="true"
          aria-label="上移"
          :disabled="index === 0"
          @click="move(index, -1)"
        >
          <KunIcon name="lucide:arrow-up" />
        </KunButton>
        <KunButton
          size="sm"
          variant="light"
          :is-icon-only="true"
          aria-label="下移"
          :disabled="index === sections.length - 1"
          @click="move(index, 1)"
        >
          <KunIcon name="lucide:arrow-down" />
        </KunButton>
        <KunButton
          size="sm"
          variant="light"
          :is-icon-only="true"
          aria-label="编辑"
          @click="openEdit(section)"
        >
          <KunIcon name="lucide:pencil" />
        </KunButton>
        <KunButton
          size="sm"
          variant="light"
          :color="section.is_archived ? 'success' : 'danger'"
          :is-icon-only="true"
          :aria-label="section.is_archived ? '恢复' : '归档'"
          @click="toggleArchive(section)"
        >
          <KunIcon
            :name="
              section.is_archived ? 'lucide:archive-restore' : 'lucide:archive'
            "
          />
        </KunButton>
      </div>
    </div>

    <AdminSectionEditModal
      v-model="isEditOpen"
      :section="editing"
      :loading="isSaving"
      @submit="saveSection"
    />
  </div>
</template>
