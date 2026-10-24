<script setup lang="ts">
import type { Section } from '#shared/utils/api/schemas'
import type { SectionForm } from './types'
import {
  SECTION_COLOR_CHOICES,
  SECTION_ICON_CHOICES
} from '~/constants/sectionIcon'

const MAX_MASCOTS = 12

const props = defineProps<{
  section: Section | null
  loading?: boolean
}>()

const emits = defineEmits<{
  submit: [data: SectionForm]
}>()

const open = defineModel<boolean>({ required: true })

const form = reactive<SectionForm>({
  label: '',
  description: '',
  icon_name: 'lucide:hash',
  color: 'default',
  banner_image_hash: '',
  mascot_image_hashes: []
})
const bannerPreview = ref('')
const mascotPreviews = ref<string[]>([])
const isUploading = ref(false)

watch(open, (isOpen) => {
  const s = props.section
  if (isOpen && s) {
    Object.assign(form, {
      label: s.label,
      description: s.description,
      icon_name: s.icon_name,
      color: s.color,
      banner_image_hash: s.banner?.hash ?? '',
      mascot_image_hashes: s.mascots.map((m) => m.hash)
    })
    bannerPreview.value = s.banner?.url ?? ''
    mascotPreviews.value = s.mascots.map((m) => m.url)
  }
})

const iconOptions = computed(() =>
  [...new Set([form.icon_name, ...SECTION_ICON_CHOICES])].map((name) => ({
    value: name,
    label: name.replace('lucide:', '')
  }))
)

const addMascots = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const files = [...(input.files ?? [])].slice(
    0,
    MAX_MASCOTS - form.mascot_image_hashes.length
  )
  isUploading.value = true
  for (const file of files) {
    const image = await uploadImage(file, 'content', file.name)
    if (image) {
      form.mascot_image_hashes.push(image.hash)
      mascotPreviews.value.push(image.url)
    }
  }
  isUploading.value = false
  input.value = ''
}

const removeMascot = (index: number) => {
  form.mascot_image_hashes.splice(index, 1)
  mascotPreviews.value.splice(index, 1)
}

const handleSubmit = () => {
  if (!form.label.trim()) {
    useMessage('请填写分区名称', 'warn')
    return
  }
  emits('submit', {
    ...form,
    mascot_image_hashes: [...form.mascot_image_hashes]
  })
}
</script>

<template>
  <KunModal
    v-model="open"
    :is-dismissable="false"
    aria-label="编辑分区"
    inner-class-name="max-w-2xl"
  >
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <h2 class="text-xl font-bold">编辑分区</h2>
      <KunInput v-model="form.label" label="分区名称" required />
      <KunTextarea
        v-model="form.description"
        label="分区介绍"
        :rows="6"
        :maxlength="500"
      />
      <div class="flex flex-wrap items-end gap-3">
        <KunSelect
          v-model="form.icon_name"
          label="图标"
          :options="iconOptions"
        />
        <KunIcon :name="form.icon_name" class="mb-2 size-5" />
        <KunSelect
          v-model="form.color"
          label="颜色"
          :options="SECTION_COLOR_CHOICES"
        />
      </div>

      <KunCoverUpload
        v-model="form.banner_image_hash"
        :preview-url="bannerPreview"
        label="分区头图 (显示在分区标题的背景上)"
      />

      <div class="space-y-2">
        <p class="text-sm font-medium">
          站娘图片 ({{ form.mascot_image_hashes.length }}/{{ MAX_MASCOTS }},
          每次进入分区随机显示一张, 建议使用透明背景的 PNG)
        </p>
        <div class="flex flex-wrap gap-2">
          <div
            v-for="(url, index) in mascotPreviews"
            :key="form.mascot_image_hashes[index]"
            class="border-default-200 relative h-24 w-20 overflow-hidden rounded-lg border"
          >
            <img :src="url" alt="" class="size-full object-contain" />
            <KunButton
              size="xs"
              color="danger"
              :is-icon-only="true"
              aria-label="移除"
              class-name="absolute top-1 right-1"
              @click="removeMascot(index)"
            >
              <KunIcon name="lucide:x" class="text-inherit" />
            </KunButton>
          </div>
          <label
            v-if="form.mascot_image_hashes.length < MAX_MASCOTS"
            :class="
              cn(
                'border-default-300 hover:bg-default-100 flex h-24 w-20 cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed text-xs',
                isUploading && 'pointer-events-none opacity-60'
              )
            "
          >
            <KunIcon name="lucide:image-plus" class="size-5" />
            {{ isUploading ? '上传中' : '添加' }}
            <input
              type="file"
              accept="image/*"
              multiple
              class="hidden"
              @change="addMascots"
            />
          </label>
        </div>
      </div>

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton type="submit" :loading="loading" :disabled="isUploading">
          保存
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
