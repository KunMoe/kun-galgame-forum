<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const ACCEPT_TYPES = 'image/png,image/jpeg,image/webp,image/gif,image/avif'
const MAX_BYTES = 4 * 1024 * 1024

const api = useApiClient()
const userStore = usePersistUserStore()

const fileInput = ref<HTMLInputElement | null>(null)
const previewUrl = ref<string | null>(null)
const pendingFile = ref<File | null>(null)
const isUploading = ref(false)

const openPicker = () => fileInput.value?.click()

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  if (!ACCEPT_TYPES.split(',').includes(file.type)) {
    useMessage('请选择 PNG / JPEG / WebP / GIF / AVIF 格式的图片', 'warn')
    target.value = ''
    return
  }
  if (file.size > MAX_BYTES) {
    useMessage('图片不能超过 4 MiB', 'warn')
    target.value = ''
    return
  }

  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = URL.createObjectURL(file)
  pendingFile.value = file
}

const clearPick = () => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = null
  pendingFile.value = null
  if (fileInput.value) fileInput.value.value = ''
}

const submit = async () => {
  if (!pendingFile.value) return
  isUploading.value = true
  const fd = new FormData()
  fd.append('file', pendingFile.value)

  const result = await settle(
    api.PUT('/me/avatar', {
      body: fd as unknown as { file?: string },
      bodySerializer: (body) => body
    })
  )
  isUploading.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('头像更新成功', 'success')
  userStore.avatar = result.data.url
  userStore.avatarMin = withImageVariant(result.data.url, '100')
  clearPick()
}
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div class="space-y-2">
      <span class="text-xl">更改头像</span>
      <p class="text-default-500 text-sm">
        头像统一由 {{ nextmoe.account }} 管理, 在
        {{ kungal.titleShort }} 直接上传图片即可生效。支持 PNG / JPEG / WebP /
        GIF / AVIF, 不超过 4 MiB。
      </p>
      <p class="text-default-500 text-sm">
        默认头像将会从
        <KunLink size="sm" :to="kungal.domain.sticker" target="_blank">
          鲲 Galgame 表情包
        </KunLink>
        中随机选取, 每一次都是不同的孩子哦, 欸嘿嘿嘿
      </p>
    </div>

    <input
      ref="fileInput"
      type="file"
      :accept="ACCEPT_TYPES"
      class="hidden"
      @change="handleFileChange"
    />

    <div class="flex items-center gap-4">
      <KunAvatar
        :user="{
          id: userStore.id,
          avatar: previewUrl ?? userStore.avatar,
          name: userStore.name
        }"
        size="lg"
        :is-navigation="false"
      />
      <div class="flex-1 text-sm">
        <p v-if="!pendingFile" class="text-default-500">
          当前头像。点击右侧按钮选择新图片。
        </p>
        <p v-else class="text-default-700">
          已选择 <strong>{{ pendingFile.name }}</strong>
          <span class="text-default-400">
            ({{ Math.round(pendingFile.size / 1024) }} KB)
          </span>
        </p>
      </div>
    </div>

    <div class="flex flex-wrap justify-end gap-2">
      <KunButton v-if="pendingFile" variant="light" @click="clearPick">
        取消
      </KunButton>
      <KunButton variant="flat" @click="openPicker">
        {{ pendingFile ? '重新选择' : '选择图片' }}
      </KunButton>
      <KunButton
        :disabled="!pendingFile || isUploading"
        :loading="isUploading"
        @click="submit"
      >
        上传并保存
      </KunButton>
    </div>
  </KunCard>
</template>
