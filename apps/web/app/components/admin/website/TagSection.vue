<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { WebsiteTag, WebsiteTagGroup } from '#shared/utils/api/schemas'
import type {
  WebsiteTagForm,
  WebsiteTagGroupForm
} from '~/components/website/modal/types'

const api = useApiClient()
const { data: tags, refresh: refreshTags } = useWebsiteTags()
const { data: groups, refresh: refreshGroups } = useWebsiteTagGroups()

const canCreate = useCan('website.create')
const canDelete = useCan('website.delete')

const UNGROUPED = 'ungrouped'

const sections = computed(() => {
  const byGroup = new Map<string, WebsiteTag[]>()
  for (const tag of tags.value ?? []) {
    const key = tag.website_tag_group_id ?? UNGROUPED
    if (!byGroup.has(key)) {
      byGroup.set(key, [])
    }
    byGroup.get(key)!.push(tag)
  }
  for (const list of byGroup.values()) {
    list.sort((a, b) => b.level - a.level)
  }

  const result: { group: WebsiteTagGroup | null; tags: WebsiteTag[] }[] = (
    groups.value ?? []
  ).map((group) => ({
    group,
    tags: byGroup.get(group.id) ?? []
  }))

  const ungrouped = byGroup.get(UNGROUPED) ?? []
  if (ungrouped.length) {
    result.push({ group: null, tags: ungrouped })
  }
  return result
})

const isTagModalOpen = ref(false)
const isGroupModalOpen = ref(false)
const isSubmitting = ref(false)
const editingTagId = ref<string | null>(null)
const tagForm = ref<WebsiteTagForm | undefined>(undefined)
const editingGroup = ref<WebsiteTagGroup | null>(null)
const groupForm = computed<WebsiteTagGroupForm | undefined>(() =>
  editingGroup.value
    ? {
        slug: editingGroup.value.slug,
        label: editingGroup.value.label,
        description: editingGroup.value.description,
        sort_order: editingGroup.value.sort_order,
        is_multi_select: editingGroup.value.is_multi_select
      }
    : undefined
)

const openCreateTag = (groupId: string | null) => {
  editingTagId.value = null
  tagForm.value = {
    slug: '',
    label: '',
    level: 0,
    description: '',
    website_tag_group_id: groupId
  }
  isTagModalOpen.value = true
}

const openEditTag = (tag: WebsiteTag) => {
  editingTagId.value = tag.id
  tagForm.value = {
    slug: tag.slug,
    label: tag.label,
    level: tag.level,
    description: tag.description,
    website_tag_group_id: tag.website_tag_group_id ?? null
  }
  isTagModalOpen.value = true
}

const handleTagSubmit = async (body: WebsiteTagForm) => {
  const target = editingTagId.value
  isSubmitting.value = true
  const result = target
    ? await settle(
        api.PATCH('/admin/website-tags/{website_tag_id}', {
          params: { path: { website_tag_id: target } },
          body
        })
      )
    : await settle(api.POST('/admin/website-tags', { body }))
  isSubmitting.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(target ? '标签已更新' : '标签已创建', 'success')
  isTagModalOpen.value = false
  await refreshTags()
}

const handleTagDelete = async (tag: WebsiteTag) => {
  const confirmed = await useComponentMessageStore().alert(
    `确定删除标签「${tag.label}」吗？`,
    '所有网站身上的这个标签会一并移除, 该操作不可撤销'
  )
  if (!confirmed) {
    return
  }

  const result = await settle(
    api.DELETE('/admin/website-tags/{website_tag_id}', {
      params: { path: { website_tag_id: tag.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('标签已删除', 'success')
  await refreshTags()
}

const openCreateGroup = () => {
  editingGroup.value = null
  isGroupModalOpen.value = true
}

const openEditGroup = (group: WebsiteTagGroup) => {
  editingGroup.value = group
  isGroupModalOpen.value = true
}

const handleGroupSubmit = async (body: WebsiteTagGroupForm) => {
  const target = editingGroup.value
  isSubmitting.value = true
  const result = target
    ? await settle(
        api.PATCH('/admin/website-tag-groups/{website_tag_group_id}', {
          params: { path: { website_tag_group_id: target.id } },
          body
        })
      )
    : await settle(api.POST('/admin/website-tag-groups', { body }))
  isSubmitting.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(target ? '分组已更新' : '分组已创建', 'success')
  isGroupModalOpen.value = false
  await refreshGroups()
}

const handleGroupDelete = async (group: WebsiteTagGroup) => {
  const confirmed = await useComponentMessageStore().alert(
    `确定删除分组「${group.label}」吗？`,
    '组内的标签不会被删除, 它们会落到「未分组」里'
  )
  if (!confirmed) {
    return
  }

  const result = await settle(
    api.DELETE('/admin/website-tag-groups/{website_tag_group_id}', {
      params: { path: { website_tag_group_id: group.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('分组已删除', 'success')
  await Promise.all([refreshGroups(), refreshTags()])
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-default-500 text-sm">
        标签决定网站的价值精算值,
        分组决定它们在创建网站表单里的排布。互斥分组只能选一个标签,
        多选分组可以任意勾选。
      </p>
      <div class="flex flex-wrap gap-3">
        <KunButton v-if="canCreate" variant="flat" @click="openCreateGroup">
          <KunIcon name="lucide:folder-plus" />
          创建分组
        </KunButton>
        <KunButton
          v-if="canCreate"
          color="primary"
          @click="openCreateTag(null)"
        >
          <KunIcon name="lucide:plus" />
          创建标签
        </KunButton>
      </div>
    </div>

    <div v-for="section in sections" :key="section.group?.id ?? UNGROUPED">
      <div class="mb-2 flex flex-wrap items-center gap-2">
        <h2 class="text-default-900 text-xl font-bold">
          {{ section.group?.label ?? '未分组' }}
        </h2>
        <span v-if="section.group" class="text-default-400 font-mono text-xs">
          {{ section.group.slug }}
        </span>
        <KunChip>{{ section.tags.length }} 个标签</KunChip>
        <KunChip v-if="section.group?.is_multi_select" color="primary">
          多选
        </KunChip>
        <KunChip v-else-if="section.group" color="secondary">互斥</KunChip>

        <template v-if="section.group">
          <KunButton
            size="sm"
            variant="light"
            :is-icon-only="true"
            @click="openEditGroup(section.group)"
          >
            <KunIcon name="lucide:pencil" />
          </KunButton>
          <KunButton
            v-if="canDelete"
            size="sm"
            variant="light"
            color="danger"
            :is-icon-only="true"
            @click="handleGroupDelete(section.group)"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
          <KunButton
            v-if="canCreate"
            size="sm"
            variant="light"
            @click="openCreateTag(section.group.id)"
          >
            <KunIcon name="lucide:plus" />
            加标签
          </KunButton>
        </template>
      </div>

      <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
        <KunCard
          v-for="tag in section.tags"
          :key="tag.id"
          :is-hoverable="false"
          :is-transparent="false"
          padding="sm"
        >
          <div class="flex items-center gap-3">
            <span
              :class="
                cn(
                  'w-12 shrink-0 text-center font-mono text-sm font-bold',
                  tag.level > 0
                    ? 'text-success-600'
                    : tag.level < 0
                      ? 'text-danger-600'
                      : 'text-default-400'
                )
              "
            >
              {{ tag.level > 0 ? '+' : '' }}{{ tag.level }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <KunLink :to="`/website-tag/${tag.slug}`" underline="none">
                  <span class="font-medium">{{ tag.label || tag.slug }}</span>
                </KunLink>
                <span class="text-default-400 font-mono text-xs">
                  {{ tag.slug }}
                </span>
              </div>
              <p v-if="tag.description" class="text-default-500 text-xs">
                {{ tag.description }}
              </p>
            </div>
            <KunButton
              size="sm"
              variant="light"
              :is-icon-only="true"
              @click="openEditTag(tag)"
            >
              <KunIcon name="lucide:pencil" />
            </KunButton>
            <KunButton
              v-if="canDelete"
              size="sm"
              variant="light"
              color="danger"
              :is-icon-only="true"
              @click="handleTagDelete(tag)"
            >
              <KunIcon name="lucide:trash-2" />
            </KunButton>
          </div>
        </KunCard>
      </div>

      <div v-if="!section.tags.length" class="text-default-400 py-3 text-sm">
        这个分组下还没有标签
      </div>
    </div>

    <KunNull v-if="!sections.length" description="还没有任何网站标签" />

    <WebsiteModalTag
      v-model="isTagModalOpen"
      :initial-data="tagForm"
      :is-editing="!!editingTagId"
      :loading="isSubmitting"
      @submit="handleTagSubmit"
    />

    <WebsiteModalTagGroup
      v-model="isGroupModalOpen"
      :initial-data="groupForm"
      :is-editing="!!editingGroup"
      :loading="isSubmitting"
      @submit="handleGroupSubmit"
    />
  </div>
</template>
