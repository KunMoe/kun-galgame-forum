<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { Section, SectionGroupKey } from '#shared/utils/api/schemas'
import type { TopicSectionGroup } from '~/composables/useTopicSections'

const api = useApiClient()
const { groups, refresh } = useTopicSections()

const editing = ref<Section | null>(null)
const isEditOpen = ref(false)
const isSaving = ref(false)

const openEdit = (section: Section) => {
  editing.value = section
  isEditOpen.value = true
}

const afterWrite = async (ok: boolean, message: string) => {
  if (ok) {
    useMessage(message, 'success')
    await refresh()
  }
}

const saveSection = async (
  body: Pick<Section, 'label' | 'description' | 'icon_name' | 'color'>
) => {
  const target = editing.value
  if (!target) {
    return
  }
  isSaving.value = true
  const result = await settle(
    api.PATCH('/admin/sections/{section}', {
      params: { path: { section: target.section } },
      body
    })
  )
  isSaving.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  isEditOpen.value = false
  await afterWrite(true, '分区已更新')
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
  if (!confirmed) {
    return
  }
  const result = await settle(
    api.PATCH('/admin/sections/{section}', {
      params: { path: { section: section.section } },
      body: { is_archived: next }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  await afterWrite(true, next ? '分区已归档' : '分区已恢复')
}

const renameGroup = async (group: TopicSectionGroup, label: string) => {
  if (!label.trim() || label === group.label) {
    return
  }
  const result = await settle(
    api.PATCH('/admin/section-groups/{group}', {
      params: { path: { group: group.key } },
      body: { label }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  await afterWrite(true, '分组已重命名')
}

const swap = <T,>(list: T[], from: number, to: number) => {
  const next = [...list]
  ;[next[from], next[to]] = [next[to]!, next[from]!]
  return next
}

const putOrder = async (
  order: { group: SectionGroupKey; sections: string[] }[]
) => {
  const result = await settle(
    api.PUT('/admin/section-order', {
      body: {
        groups: order.map((g) => ({
          group: g.group,
          sections: g.sections as Section['section'][]
        }))
      }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  await afterWrite(true, '顺序已更新')
}

const currentOrder = () =>
  groups.value.map((g) => ({
    group: g.key,
    sections: g.sections.map((s) => s.section as string)
  }))

const moveGroup = (index: number, delta: number) =>
  putOrder(swap(currentOrder(), index, index + delta))

const moveSection = (groupIndex: number, index: number, delta: number) => {
  const order = currentOrder()
  const group = order[groupIndex]!
  group.sections = swap(group.sections, index, index + delta)
  return putOrder(order)
}
</script>

<template>
  <div class="space-y-6">
    <KunCard
      v-for="(group, gi) in groups"
      :key="group.key"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <div class="flex flex-wrap items-center gap-2">
        <KunInput
          :model-value="group.label"
          aria-label="分组名称"
          class-name="max-w-48"
          size="sm"
          @change="
            (e: Event) =>
              renameGroup(group, (e.target as HTMLInputElement).value)
          "
        />
        <span class="text-default-400 font-mono text-xs">{{ group.key }}</span>
        <div class="ml-auto flex gap-1">
          <KunButton
            size="sm"
            variant="light"
            :is-icon-only="true"
            aria-label="分组上移"
            :disabled="gi === 0"
            @click="moveGroup(gi, -1)"
          >
            <KunIcon name="lucide:arrow-up" />
          </KunButton>
          <KunButton
            size="sm"
            variant="light"
            :is-icon-only="true"
            aria-label="分组下移"
            :disabled="gi === groups.length - 1"
            @click="moveGroup(gi, 1)"
          >
            <KunIcon name="lucide:arrow-down" />
          </KunButton>
        </div>
      </div>

      <div
        v-for="(section, si) in group.sections"
        :key="section.section"
        :class="
          cn(
            'border-default-200 flex items-start gap-3 rounded-xl border p-3',
            section.is_archived && 'opacity-60'
          )
        "
      >
        <KunIcon :name="section.icon_name" class="mt-1 shrink-0" />
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
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
              不进入主页动态
            </KunChip>
            <KunChip v-if="section.is_archived" size="sm">已归档</KunChip>
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
            :disabled="si === 0"
            @click="moveSection(gi, si, -1)"
          >
            <KunIcon name="lucide:arrow-up" />
          </KunButton>
          <KunButton
            size="sm"
            variant="light"
            :is-icon-only="true"
            aria-label="下移"
            :disabled="si === group.sections.length - 1"
            @click="moveSection(gi, si, 1)"
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
                section.is_archived
                  ? 'lucide:archive-restore'
                  : 'lucide:archive'
              "
            />
          </KunButton>
        </div>
      </div>
    </KunCard>

    <AdminSectionEditModal
      v-model="isEditOpen"
      :section="editing"
      :loading="isSaving"
      @submit="saveSection"
    />
  </div>
</template>
