<script setup lang="ts">
import type { WebsiteTag } from '#shared/utils/api/schemas'

const props = defineProps<{
  tags: WebsiteTag[]
  tagIds: string[]
}>()

const emits = defineEmits<{
  updateIds: [ids: string[]]
}>()

const { data: groups } = useWebsiteTagGroups()

const UNGROUPED = 'ungrouped'

const groupedTags = computed(() => {
  const byGroup = new Map<string, WebsiteTag[]>()
  for (const tag of props.tags ?? []) {
    const key = tag.website_tag_group_id ?? UNGROUPED
    if (!byGroup.has(key)) {
      byGroup.set(key, [])
    }
    byGroup.get(key)!.push(tag)
  }

  const sections = (groups.value ?? [])
    .map((group) => ({
      id: group.id,
      label: group.label || group.slug,
      multiSelect: group.is_multi_select,
      tags: byGroup.get(group.id) ?? []
    }))
    .filter((section) => section.tags.length > 0)

  const ungrouped = byGroup.get(UNGROUPED) ?? []
  if (ungrouped.length) {
    sections.push({
      id: UNGROUPED,
      label: '未分组',
      multiSelect: true,
      tags: ungrouped
    })
  }

  return sections
})

const MAX_TAGS = 20

const toggleExclusive = (tagId: string, sectionTags: WebsiteTag[]) => {
  if (props.tagIds.includes(tagId)) {
    emits(
      'updateIds',
      props.tagIds.filter((id) => id !== tagId)
    )
    return
  }
  const sectionIds = sectionTags.map((tag) => tag.id)
  emits('updateIds', [
    ...props.tagIds.filter((id) => !sectionIds.includes(id)),
    tagId
  ])
}

const toggleMultiple = (checked: boolean, tagId: string) => {
  if (!checked) {
    emits(
      'updateIds',
      props.tagIds.filter((id) => id !== tagId)
    )
    return
  }
  if (props.tagIds.includes(tagId)) {
    return
  }
  if (props.tagIds.length >= MAX_TAGS) {
    useMessage(`您最多选择 ${MAX_TAGS} 个网站标签`, 'warn')
    return
  }
  emits('updateIds', [...props.tagIds, tagId])
}
</script>

<template>
  <div class="space-y-4">
    <fieldset
      v-for="section in groupedTags"
      :key="section.id"
      class="border-default-200 rounded-md border p-3"
    >
      <legend class="text-default-700 px-2 text-sm font-semibold">
        {{ section.label }}
        <span v-if="!section.multiSelect" class="text-default-400 font-normal">
          (单选)
        </span>
      </legend>
      <div class="flex flex-wrap gap-x-4 gap-y-2 pt-2">
        <KunCheckBox
          v-for="tag in section.tags"
          :id="tag.slug"
          :key="tag.id"
          :model-value="tagIds.includes(tag.id)"
          :label="tag.label || tag.slug"
          :value="tag.id"
          class-name="w-full p-1 hover:bg-default-100 rounded"
          @update:model-value="
            (checked) =>
              section.multiSelect
                ? toggleMultiple(checked, tag.id)
                : toggleExclusive(tag.id, section.tags)
          "
        />
      </div>
    </fieldset>

    <KunNull
      v-if="!groupedTags.length"
      description="暂无可选标签, 请先在标签管理页面创建"
    />
  </div>
</template>
