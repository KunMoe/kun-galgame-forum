<script setup lang="ts">
import type { Section } from '#shared/utils/api/schemas'
import {
  SECTION_COLOR_CHOICES,
  SECTION_ICON_CHOICES
} from '~/constants/sectionIcon'

type SectionForm = Pick<
  Section,
  'label' | 'description' | 'icon_name' | 'color'
>

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
  color: 'default'
})

watch(open, (isOpen) => {
  if (isOpen && props.section) {
    Object.assign(form, {
      label: props.section.label,
      description: props.section.description,
      icon_name: props.section.icon_name,
      color: props.section.color
    })
  }
})

const iconOptions = computed(() =>
  [...new Set([form.icon_name, ...SECTION_ICON_CHOICES])].map((name) => ({
    value: name,
    label: name.replace('lucide:', '')
  }))
)

const handleSubmit = () => {
  if (!form.label.trim()) {
    useMessage('请填写分区名称', 'warn')
    return
  }
  emits('submit', { ...form })
}
</script>

<template>
  <KunModal
    v-model="open"
    :is-dismissable="false"
    aria-label="编辑分区"
    inner-class-name="max-w-lg"
  >
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <h2 class="text-xl font-bold">编辑分区</h2>
      <KunInput v-model="form.label" label="分区名称" required />
      <KunTextarea
        v-model="form.description"
        label="分区说明"
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
      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton type="submit" :loading="loading">保存</KunButton>
      </div>
    </form>
  </KunModal>
</template>
