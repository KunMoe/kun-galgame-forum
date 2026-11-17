<script setup lang="ts">
import { formatFileSize } from '#shared/utils/format'
const props = defineProps<{ options: AgentUploadOption[]; full: boolean }>()
const emit = defineEmits<{
  pick: [option: AgentUploadOption, files: FileList]
}>()

const picker = useTemplateRef<HTMLInputElement>('picker')
const menu = useTemplateRef<{ close: () => void }>('menu')
const current = ref<AgentUploadOption | null>(null)

const choose = async (option: AgentUploadOption) => {
  menu.value?.close()
  current.value = option
  await nextTick()
  picker.value?.click()
}

const onChange = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (current.value && input.files?.length)
    emit('pick', current.value, input.files)
  input.value = ''
}
</script>

<template>
  <KunPopover
    ref="menu"
    position="top-start"
    inner-class="p-1 min-w-48"
    aria-label="上传文件"
  >
    <template #trigger>
      <KunButton
        variant="light"
        size="sm"
        is-icon-only
        aria-label="上传文件"
        :disabled="props.full"
      >
        <KunIcon name="lucide:paperclip" class="size-5" />
      </KunButton>
    </template>
    <div class="flex flex-col">
      <button
        v-for="o in options"
        :key="o.kind"
        type="button"
        class="hover:bg-default-100 rounded-kun-md flex w-full items-center gap-2 px-2 py-2 text-left text-sm"
        @click="choose(o)"
      >
        <KunIcon :name="`lucide:${o.icon}`" class="size-4 shrink-0" />
        <span class="flex-1">{{ o.label }}</span>
        <span class="text-default-400 text-xs tabular-nums">
          ≤ {{ formatFileSize(o.max_bytes) }}
        </span>
      </button>
    </div>
  </KunPopover>
  <input
    ref="picker"
    type="file"
    multiple
    :accept="current?.accept"
    class="hidden"
    tabindex="-1"
    aria-hidden="true"
    @change="onChange"
  />
</template>
