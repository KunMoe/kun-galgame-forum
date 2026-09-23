<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { AdminWebsite, Website } from '#shared/utils/api/schemas'
import type { WebsiteForm } from './modal/types'
import { changedWebsiteFields, websiteFormOf } from '~/utils/websiteForm'

const props = defineProps<{
  website: Website
}>()

const emits = defineEmits<{
  refresh: []
}>()

const api = useApiClient()
const { id } = usePersistUserStore()
const utmLink = useUtmLink()

const isLiked = ref(props.website.viewer?.has_liked ?? false)
const likeCount = ref(props.website.like_count)
watch(
  () => props.website,
  (w) => {
    isLiked.value = w.viewer?.has_liked ?? false
    likeCount.value = w.like_count
  }
)

const slotPath = (slot: 'like' | 'favorite') =>
  slot === 'like'
    ? ('/websites/{website_host}/like' as const)
    : ('/websites/{website_host}/favorite' as const)

const setSlot = async (slot: 'like' | 'favorite', next: boolean) => {
  const options = { params: { path: { website_host: props.website.host } } }
  const result = await settle(
    next
      ? api.PUT(slotPath(slot), options)
      : api.DELETE(slotPath(slot), options)
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return null
  }
  return result.data
}

const likePending = ref(false)
const onLike = async (next: boolean) => {
  if (!id) {
    useAuthModal().open()
    isLiked.value = !next
    likeCount.value += next ? -1 : 1
    return
  }
  likePending.value = true
  const engagement = await setSlot('like', next)
  likePending.value = false
  if (!engagement) {
    isLiked.value = !next
    likeCount.value += next ? -1 : 1
    return
  }
  isLiked.value = engagement.viewer?.has_liked ?? next
  likeCount.value = engagement.like_count
  useMessage(next ? '点赞网站成功!' : '取消点赞成功!', 'success')
}

const onFavorite = async (next: boolean) =>
  Boolean(await setSlot('favorite', next))

const showWebsiteModal = ref(false)
const editSource = ref<AdminWebsite | null>(null)
const editForm = computed(() =>
  editSource.value ? websiteFormOf(editSource.value) : undefined
)
const updating = ref(false)
const deleting = ref(false)

const handleOpenUpdateModal = async () => {
  const result = await settle(
    api.GET('/admin/websites/{website_id}', {
      params: { path: { website_id: props.website.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  editSource.value = result.data
  showWebsiteModal.value = true
}

const handleUpdate = async (data: WebsiteForm) => {
  if (!editForm.value || updating.value) {
    return
  }
  const body = changedWebsiteFields(editForm.value, data)
  updating.value = true
  const result = await settle(
    api.PATCH('/admin/websites/{website_id}', {
      params: { path: { website_id: props.website.id } },
      body
    })
  )
  updating.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('重新编辑成功', 'success')
  showWebsiteModal.value = false
  if (result.data.host !== props.website.host) {
    await navigateTo(`/website/${result.data.host}`)
    return
  }
  emits('refresh')
}

const handleDelete = async () => {
  if (deleting.value) {
    return
  }
  const confirmed = await useComponentMessageStore().alert(
    '您确定删除这个网站吗？',
    '这将会删除这个网站的所有信息, 该操作不可撤销'
  )
  if (!confirmed) {
    return
  }
  deleting.value = true
  const result = await settle(
    api.DELETE('/admin/websites/{website_id}', {
      params: { path: { website_id: props.website.id } }
    })
  )
  deleting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除网站成功', 'success')
  await navigateTo('/website')
}
</script>

<template>
  <div class="flex flex-wrap gap-3">
    <KunTooltip text="点赞">
      <span class="flex">
        <KunReaction
          v-model="isLiked"
          v-model:count="likeCount"
          :disabled="likePending"
          size="lg"
          icon="lucide:thumbs-up"
          color="secondary"
          label="点赞"
          @change="onLike"
        />
      </span>
    </KunTooltip>

    <FavoriteToggle
      :favorited="website.viewer?.has_favorited ?? false"
      :count="website.favorite_count"
      :action="onFavorite"
      :messages="['收藏网站成功!', '取消收藏成功!']"
      size="lg"
    />

    <KunTooltip v-if="website.viewer?.can_edit" text="编辑">
      <KunButton
        :is-icon-only="true"
        size="lg"
        variant="light"
        class-name="gap-1"
        @click="handleOpenUpdateModal"
      >
        <KunIcon name="lucide:pen" />
      </KunButton>
    </KunTooltip>

    <WebsiteModalWebsite
      v-model="showWebsiteModal"
      :initial-data="editForm"
      :is-editing="true"
      :icon-preview-url="
        editSource?.icon?.url ?? editSource?.external_icon_url ?? ''
      "
      :has-external-icon="!!editSource?.external_icon_url"
      :loading="updating"
      @submit="handleUpdate"
    />

    <KunTooltip v-if="website.viewer?.can_delete" text="删除">
      <KunButton
        :is-icon-only="true"
        size="lg"
        color="danger"
        variant="light"
        class-name="gap-1"
        :loading="deleting"
        @click="handleDelete"
      >
        <KunIcon name="lucide:trash-2" />
      </KunButton>
    </KunTooltip>

    <KunButton
      target="_blank"
      :href="utmLink(`https://${website.host}`)"
      class-name="ml-auto"
    >
      访问网站
    </KunButton>
  </div>
</template>
