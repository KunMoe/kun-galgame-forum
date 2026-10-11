<script setup lang="ts">
import type { ToolsetPendingUpload } from '~/composables/useToolsetResumeUploads'

defineProps<{
  finished: ToolsetPendingUpload[]
}>()

const emits = defineEmits<{
  use: [record: ToolsetPendingUpload]
  discard: [uploadId: string]
}>()

const confirmDiscard = async (record: ToolsetPendingUpload) => {
  const ok = await useComponentMessageStore().alert(
    '确定放弃这个已上传的文件吗?',
    '放弃后需要重新上传才能发布, 已经用掉的今日上传额度不会退回。'
  )
  if (ok) {
    emits('discard', record.id)
  }
}
</script>

<template>
  <div class="space-y-2">
    <div
      v-for="item in finished"
      :key="item.id"
      class="border-success/30 bg-success/10 flex flex-col gap-2 rounded-lg border p-3 text-sm sm:flex-row sm:items-center"
    >
      <div class="flex min-w-0 flex-1 items-start gap-2">
        <KunIcon
          name="lucide:file-check"
          class="text-success mt-0.5 shrink-0"
        />
        <span class="text-default-700 min-w-0 break-all">
          你有一个已上传完成但尚未发布的文件「{{ item.name }}」({{
            formatFileSize(item.size)
          }}), 可以直接用它发布, 不必重新上传
        </span>
      </div>
      <div class="flex shrink-0 items-center justify-end gap-1">
        <KunButton size="sm" variant="flat" @click="emits('use', item)">
          用它发布
        </KunButton>
        <KunButton
          size="sm"
          variant="light"
          color="danger"
          @click="confirmDiscard(item)"
        >
          放弃
        </KunButton>
      </div>
    </div>
  </div>
</template>
