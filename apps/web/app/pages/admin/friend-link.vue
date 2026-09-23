<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { FriendLink, FriendLinkCategory } from '#shared/utils/api/schemas'
import type { FriendLinkSubmit } from '~/components/admin/friend-link/type'
import { FRIEND_LINK_CATEGORIES } from '~/constants/friendLink'

definePageMeta({
  middleware: 'permission',
  permissions: ['friend_link.create', 'friend_link.edit', 'friend_link.delete']
})

useKunDisableSeo('友链管理')

const api = useApiClient()
const { links, refresh } = await useFriendLinks()

const shelf = (category: FriendLinkCategory) =>
  links.value.filter((link) => link.friend_link_category === category)

const isModalOpen = ref(false)
const editing = ref<FriendLink | null>(null)
const modalCategory = ref<FriendLinkCategory>('galgame')

const openAdd = (category: FriendLinkCategory) => {
  editing.value = null
  modalCategory.value = category
  isModalOpen.value = true
}

const openEdit = (link: FriendLink) => {
  editing.value = link
  isModalOpen.value = true
}

const handleSubmit = async (submit: FriendLinkSubmit) => {
  const result = await settle(
    submit.id === null
      ? api.POST('/admin/friend-links', { body: submit.body })
      : api.PATCH('/admin/friend-links/{friend_link_id}', {
          params: { path: { friend_link_id: submit.id } },
          body: submit.body
        })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(submit.id ? '友链已更新' : '友链已添加', 'success')
  await refresh()
}

const handleRemove = async (link: FriendLink) => {
  const confirmed = await useComponentMessageStore().alert(
    `确定删除友链「${link.title}」吗？`
  )
  if (!confirmed) return

  const result = await settle(
    api.DELETE('/admin/friend-links/{friend_link_id}', {
      params: { path: { friend_link_id: link.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('友链已删除', 'success')
  await refresh()
}

const handleReorder = async (category: FriendLinkCategory, ids: string[]) => {
  const result = await settle(
    api.PUT('/admin/friend-link-order', {
      body: { friend_link_category: category, friend_link_ids: ids }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    await refresh()
  }
}
</script>

<template>
  <div class="w-full space-y-6">
    <KunHeader
      name="友链管理"
      description="增删改友链, 拖拽左侧手柄可调整每个分类内的展示顺序"
    />

    <AdminFriendLinkSection
      v-for="category in FRIEND_LINK_CATEGORIES"
      :key="category.key"
      :category="category.key"
      :label="category.label"
      :links="shelf(category.key)"
      @add="openAdd"
      @edit="openEdit"
      @remove="handleRemove"
      @reorder="handleReorder"
    />

    <AdminFriendLinkModal
      v-model="isModalOpen"
      :initial-data="editing"
      :default-category="modalCategory"
      @submit="handleSubmit"
    />
  </div>
</template>
