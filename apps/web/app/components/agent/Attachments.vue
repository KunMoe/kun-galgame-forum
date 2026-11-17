<script setup lang="ts">
import { agentPath } from '#shared/utils/api/agent'
import type { AgentAttachment } from '~/composables/useAgentAttachments'

defineProps<{ items: AgentAttachment[] }>()
const emit = defineEmits<{ remove: [key: string] }>()

const origin = useRuntimeConfig().public.apiBaseUrl as string
const srcOf = (a: AgentAttachment) =>
  a.upload
    ? agentPath(origin, `/uploads/${encodeURIComponent(a.upload.id)}`)
    : ''
</script>

<template>
  <div class="flex flex-wrap items-end gap-2 px-1 pt-1">
    <template v-for="a in items" :key="a.key">
      <div
        v-if="a.kind === 'image'"
        class="border-default-200 bg-default-100 rounded-kun-md relative size-16 overflow-hidden border"
      >
        <img
          v-if="a.upload"
          :src="srcOf(a)"
          :alt="a.filename"
          class="size-full object-cover"
        />
        <div v-else class="flex size-full items-center justify-center">
          <KunIcon
            name="lucide:loader-circle"
            class="text-default-400 size-5 animate-spin"
          />
        </div>
        <button
          type="button"
          :aria-label="`移除「${a.filename}」`"
          class="bg-content1/90 rounded-kun-md absolute top-0.5 right-0.5 flex size-5 items-center justify-center"
          @click="emit('remove', a.key)"
        >
          <KunIcon name="lucide:x" class="size-3.5" />
        </button>
      </div>
      <div v-else class="flex flex-col gap-0.5">
        <KunChip
          size="sm"
          variant="flat"
          closable
          @close="emit('remove', a.key)"
        >
          <KunIcon
            :name="
              a.status === 'uploading'
                ? 'lucide:loader-circle'
                : a.kind === 'document'
                  ? 'lucide:file'
                  : 'lucide:file-text'
            "
            :class="cn('size-3.5', a.status === 'uploading' && 'animate-spin')"
          />
          {{ a.filename }}
        </KunChip>
        <span v-if="a.upload?.truncated" class="text-warning-600 px-1 text-xs">
          过长，只读了开头
        </span>
      </div>
    </template>
  </div>
</template>
