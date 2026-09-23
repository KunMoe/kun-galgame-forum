<script setup lang="ts">
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'

const MAX_COVERS = 9

const { coverImages } = useTopicEditorStore()

const isUploading = ref(false)
const progressText = ref('上传中')

const remaining = computed(() => MAX_COVERS - coverImages.value.length)

const uploadFiles = async (files: File[]) => {
  if (isUploading.value) return
  if (remaining.value <= 0) {
    useMessage(`封面图最多 ${MAX_COVERS} 张`, 'warn')
    return
  }

  const images = files.filter((f) => f.type.startsWith('image/'))
  if (images.length < files.length) {
    useMessage('只能上传图片文件', 'warn')
  }
  const picked = images.slice(0, remaining.value)
  if (!picked.length) return

  isUploading.value = true
  let done = 0
  try {
    const tokens = await Promise.all(
      picked.map(async (image) => {
        const uploaded = await uploadImage(image, 'content', image.name)
        done++
        progressText.value = `上传中 ${done}/${picked.length}`
        return uploaded ? imageToken(uploaded.hash) : null
      })
    )

    const next = [...coverImages.value]
    for (const token of tokens) {
      if (token && !next.includes(token)) {
        next.push(token)
      }
    }
    coverImages.value = next.slice(0, MAX_COVERS)
  } catch {
    useMessage('封面图上传失败, 请重试', 'error')
  } finally {
    isUploading.value = false
    progressText.value = '上传中'
  }
}

const handleRemove = (index: number) => {
  coverImages.value = coverImages.value.filter((_, i) => i !== index)
}

const handleShift = (index: number, dir: -1 | 1) => {
  const target = index + dir
  if (target < 0 || target >= coverImages.value.length) return
  const next = [...coverImages.value]
  const tmp = next[index]!
  next[index] = next[target]!
  next[target] = tmp
  coverImages.value = next
}
</script>

<template>
  <div class="space-y-4">
    <h3 class="flex items-center gap-2 text-lg font-semibold">
      <Icon name="lucide:images" class="h-5 w-5" />
      封面图
      <span class="text-default-500 text-sm font-normal">
        (已添加 {{ coverImages.length }}/{{ MAX_COVERS }})
      </span>
    </h3>

    <div class="grid grid-cols-3 gap-3">
      <div
        v-for="(token, idx) in coverImages"
        :key="token"
        class="group border-default-200 relative overflow-hidden rounded-lg border"
      >
        <img
          :src="imageTokenUrl(token)"
          alt="封面图"
          loading="lazy"
          class="aspect-video w-full object-cover"
        />

        <div
          class="absolute top-1.5 right-1.5 flex gap-1 opacity-100 transition-opacity sm:opacity-0 sm:group-focus-within:opacity-100 sm:group-hover:opacity-100"
        >
          <KunTooltip text="前移">
            <KunButton
              :is-icon-only="true"
              size="sm"
              variant="solid"
              color="default"
              :disabled="idx === 0"
              @click="handleShift(idx, -1)"
            >
              <KunIcon name="lucide:chevron-left" />
            </KunButton>
          </KunTooltip>
          <KunTooltip text="后移">
            <KunButton
              :is-icon-only="true"
              size="sm"
              variant="solid"
              color="default"
              :disabled="idx === coverImages.length - 1"
              @click="handleShift(idx, 1)"
            >
              <KunIcon name="lucide:chevron-right" />
            </KunButton>
          </KunTooltip>
          <KunTooltip text="移除">
            <KunButton
              :is-icon-only="true"
              size="sm"
              variant="solid"
              color="danger"
              @click="handleRemove(idx)"
            >
              <KunIcon name="lucide:trash-2" />
            </KunButton>
          </KunTooltip>
        </div>

        <KunChip
          v-if="idx === 0"
          size="sm"
          color="primary"
          variant="solid"
          class-name="pointer-events-none absolute bottom-1.5 left-1.5"
        >
          主封面
        </KunChip>
      </div>

      <EditGalgamePrImageDropzone
        v-if="remaining > 0"
        label="添加封面"
        :is-uploading="isUploading"
        :progress-text="progressText"
        @files="uploadFiles"
      />
    </div>

    <p class="text-default-500 text-sm">
      可选, 最多 9 张。用箭头调整顺序, 第一张为主封面。封面会显示在首页 /
      列表的话题卡片上。
    </p>
  </div>
</template>
