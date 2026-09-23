<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { WebsiteCategory } from '#shared/utils/api/schemas'
import type { WebsiteCategoryForm } from '~/components/website/modal/types'

const api = useApiClient()
const { data, refresh } = useWebsiteCategories()

const canCreate = useCan('website.create')
const canDelete = useCan('website.delete')

const isModalOpen = ref(false)
const isSubmitting = ref(false)
const editing = ref<WebsiteCategory | null>(null)
const initialData = computed<WebsiteCategoryForm | undefined>(() =>
  editing.value
    ? {
        slug: editing.value.slug,
        label: editing.value.label,
        description: editing.value.description,
        sort_order: editing.value.sort_order
      }
    : undefined
)

const openCreate = () => {
  editing.value = null
  isModalOpen.value = true
}

const openEdit = (category: WebsiteCategory) => {
  editing.value = category
  isModalOpen.value = true
}

const handleSubmit = async (body: WebsiteCategoryForm) => {
  const target = editing.value
  isSubmitting.value = true
  const result = target
    ? await settle(
        api.PATCH('/admin/website-categories/{website_category_id}', {
          params: { path: { website_category_id: target.id } },
          body
        })
      )
    : await settle(api.POST('/admin/website-categories', { body }))
  isSubmitting.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(target ? '分类已更新' : '分类已创建', 'success')
  isModalOpen.value = false
  await refresh()
}

const handleDelete = async (category: WebsiteCategory) => {
  const confirmed = await useComponentMessageStore().alert(
    `确定删除分类「${category.label}」吗？`,
    '只有空分类可以删除, 该操作不可撤销'
  )
  if (!confirmed) {
    return
  }

  const result = await settle(
    api.DELETE('/admin/website-categories/{website_category_id}', {
      params: { path: { website_category_id: category.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('分类已删除', 'success')
  await refresh()
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-default-500 text-sm">
        分类标识是 /website-category/:name 的一部分,
        排序决定它在资料库页面的先后。
      </p>
      <KunButton v-if="canCreate" color="primary" @click="openCreate">
        <KunIcon name="lucide:plus" />
        创建分类
      </KunButton>
    </div>

    <KunCard
      v-for="category in data ?? []"
      :key="category.id"
      :is-hoverable="false"
      :is-transparent="false"
      padding="sm"
    >
      <div class="flex items-center gap-3">
        <span
          class="text-default-400 w-10 shrink-0 text-center font-mono text-sm"
        >
          {{ category.sort_order }}
        </span>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <KunLink
              :to="`/website-category/${category.slug}`"
              underline="none"
            >
              <span class="font-medium">{{ category.label }}</span>
            </KunLink>
            <span class="text-default-400 font-mono text-xs">
              {{ category.slug }}
            </span>
            <KunChip>{{ category.website_count }} 个网站</KunChip>
          </div>
          <p v-if="category.description" class="text-default-500 text-xs">
            {{ category.description }}
          </p>
        </div>
        <KunButton
          size="sm"
          variant="light"
          :is-icon-only="true"
          @click="openEdit(category)"
        >
          <KunIcon name="lucide:pencil" />
        </KunButton>
        <KunButton
          v-if="canDelete"
          size="sm"
          variant="light"
          color="danger"
          :is-icon-only="true"
          :disabled="category.website_count > 0"
          @click="handleDelete(category)"
        >
          <KunIcon name="lucide:trash-2" />
        </KunButton>
      </div>
    </KunCard>

    <KunNull v-if="!data?.length" description="还没有任何网站分类" />

    <WebsiteModalCategory
      v-model="isModalOpen"
      :initial-data="initialData"
      :is-editing="!!editing"
      :loading="isSubmitting"
      @submit="handleSubmit"
    />
  </div>
</template>
