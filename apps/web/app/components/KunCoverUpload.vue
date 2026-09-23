<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: string
    previewUrl?: string
    label?: string
  }>(),
  {
    previewUrl: '',
    label: '封面'
  }
)

const emits = defineEmits<{
  'update:modelValue': [value: string]
}>()

const previewUrl = ref(props.previewUrl)
watch(
  () => props.previewUrl,
  (url) => {
    previewUrl.value = url
  }
)

const isUploading = ref(false)

const handleFileChange = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) {
    return
  }

  isUploading.value = true
  const image = await uploadImage(file, 'content', file.name)
  isUploading.value = false
  input.value = ''

  if (image) {
    emits('update:modelValue', image.hash)
    previewUrl.value = image.url
    useMessage('封面上传成功', 'success')
  }
}

const handleRemove = () => {
  emits('update:modelValue', '')
  previewUrl.value = ''
}
</script>

<template>
  <div>
    <label class="mb-1 block text-sm font-medium">{{ label }}</label>
    <div class="flex items-start gap-3">
      <KunImage
        v-if="previewUrl"
        :src="previewUrl"
        class="border-default-200 h-20 w-32 shrink-0 rounded-md border object-cover"
      />
      <div class="flex flex-1 flex-col gap-2">
        <label
          :class="
            cn(
              'border-default-200 bg-default-100 hover:bg-default-200 inline-flex w-fit cursor-pointer items-center gap-2 rounded-md border px-3 py-2 text-sm transition-colors',
              isUploading && 'pointer-events-none opacity-60'
            )
          "
        >
          <KunIcon name="lucide:image-up" />
          <span>{{ isUploading ? '上传中...' : '选择图片' }}</span>
          <input
            type="file"
            accept="image/*"
            class="hidden"
            :disabled="isUploading"
            @change="handleFileChange"
          />
        </label>
        <KunButton
          v-if="previewUrl || modelValue"
          variant="light"
          color="danger"
          size="sm"
          class-name="w-fit"
          @click="handleRemove"
        >
          移除
        </KunButton>
      </div>
    </div>
  </div>
</template>
