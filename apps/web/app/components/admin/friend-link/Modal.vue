<script setup lang="ts">
import type {
  FriendLink,
  FriendLinkCategory,
  FriendLinkCreate,
  FriendLinkPatch
} from '#shared/utils/api/schemas'
import {
  FRIEND_LINK_CATEGORY_OPTIONS,
  FRIEND_LINK_STATE_OPTIONS
} from '~/constants/friendLink'
import type { FriendLinkSubmit } from './type'

const props = defineProps<{
  modelValue: boolean
  initialData?: FriendLink | null
  defaultCategory?: FriendLinkCategory
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: FriendLinkSubmit]
}>()

const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emits('update:modelValue', v)
})
const isEditing = computed(() => !!props.initialData?.id)

type FriendLinkForm = Required<
  Omit<FriendLinkCreate, 'banner_image_hash' | 'description' | 'state'>
> & {
  description: string
  banner_image_hash: string
  state: NonNullable<FriendLinkCreate['state']>
}

const getInitial = (): FriendLinkForm => {
  const d = props.initialData
  return {
    friend_link_category:
      d?.friend_link_category ?? props.defaultCategory ?? 'galgame',
    title: d?.title ?? '',
    url: d?.url ?? '',
    description: d?.description ?? '',
    banner_image_hash: d?.banner?.hash ?? '',
    state: d?.state ?? 'normal'
  }
}

const form = reactive<FriendLinkForm>(getInitial())
watch(
  () => props.modelValue,
  (open) => {
    if (open) Object.assign(form, getInitial())
  }
)

const initialBannerUrl = computed(() => props.initialData?.banner?.url ?? '')

const changedFields = (): FriendLinkPatch => {
  const initial = getInitial()
  const patch: FriendLinkPatch = {}
  for (const key of Object.keys(form) as (keyof FriendLinkForm)[]) {
    if (form[key] !== initial[key]) {
      Object.assign(patch, { [key]: form[key] })
    }
  }
  return patch
}

const handleSubmit = () => {
  if (!form.title.trim()) {
    useMessage('请填写友链名称', 'warn')
    return
  }
  if (!/^https?:\/\//.test(form.url.trim())) {
    useMessage('友链地址必须以 http:// 或 https:// 开头', 'warn')
    return
  }
  const id = props.initialData?.id
  if (id) {
    const body = changedFields()
    if (Object.keys(body).length) {
      emits('submit', { id, body })
    }
  } else {
    emits('submit', { id: null, body: { ...form, url: form.url.trim() } })
  }
  isOpen.value = false
}
</script>

<template>
  <KunModal
    :is-dismissable="false"
    v-model="isOpen"
    inner-class-name="max-w-2xl"
  >
    <form @submit.prevent>
      <h2 class="mb-4 text-xl font-bold">
        {{ isEditing ? '编辑友链' : '添加友链' }}
      </h2>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <KunInput v-model="form.title" label="名称" required />
        <KunInput
          v-model="form.url"
          label="链接 (URL)"
          required
          placeholder="https://..."
        />
        <KunSelect
          v-model="form.friend_link_category"
          label="分类"
          :options="FRIEND_LINK_CATEGORY_OPTIONS"
        />
        <KunSelect
          v-model="form.state"
          label="状态"
          :options="FRIEND_LINK_STATE_OPTIONS"
        />
        <KunTextarea
          v-model="form.description"
          label="描述"
          auto-grow
          show-char-count
          :maxlength="500"
          class-name="md:col-span-2"
        />

        <div class="md:col-span-2">
          <KunCoverUpload
            v-model="form.banner_image_hash"
            :preview-url="initialBannerUrl"
            label="图标 / Banner"
          />
        </div>
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <KunButton variant="light" color="danger" @click="isOpen = false">
          取消
        </KunButton>
        <KunButton color="primary" @click="handleSubmit">
          {{ isEditing ? '保存' : '添加' }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
