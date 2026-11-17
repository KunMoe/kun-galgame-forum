<script setup lang="ts">
import { agentPath } from '#shared/utils/api/agent'
import type { AgentFilePart } from '~/utils/agentUpload'

const props = defineProps<{ files: AgentFilePart[] }>()

const origin = useRuntimeConfig().public.apiBaseUrl as string
const srcOf = (f: AgentFilePart) => {
  const id = agentUploadId(f)
  return id ? agentPath(origin, `/uploads/${encodeURIComponent(id)}`) : ''
}

const images = computed(() => props.files.filter(isAgentImage))
const others = computed(() => props.files.filter((f) => !isAgentImage(f)))
</script>

<template>
  <div class="flex flex-col items-end gap-1.5">
    <div v-if="images.length" class="flex flex-wrap justify-end gap-1.5">
      <KunLightboxGallery>
        <KunLightboxGalleryItem
          v-for="(f, i) in images"
          :key="agentUploadId(f)"
          v-slot="{ open }"
          :src="srcOf(f)"
          :alt="f.filename || `图片 ${i + 1}`"
          :wrap="false"
        >
          <button
            type="button"
            class="border-default-200 bg-default-100 rounded-kun-md size-24 cursor-zoom-in overflow-hidden border"
            :aria-label="`查看「${f.filename || `图片 ${i + 1}`}」大图`"
            @click="open"
          >
            <img
              :src="srcOf(f)"
              :alt="f.filename"
              loading="lazy"
              class="size-full object-cover"
            />
          </button>
        </KunLightboxGalleryItem>
      </KunLightboxGallery>
    </div>
    <div v-if="others.length" class="flex flex-wrap justify-end gap-1.5">
      <component
        :is="srcOf(f) ? 'a' : 'div'"
        v-for="(f, i) in others"
        :key="agentUploadId(f) || i"
        :href="srcOf(f) || undefined"
        :target="srcOf(f) ? '_blank' : undefined"
        :rel="srcOf(f) ? 'noopener noreferrer' : undefined"
        :title="srcOf(f) ? '查看模型读到的文字' : undefined"
        class="border-default-200 bg-content1 rounded-kun-md hover:border-primary/50 flex max-w-60 items-center gap-2 border px-2.5 py-1.5 text-sm transition-colors"
      >
        <KunIcon
          :name="
            f.mediaType.startsWith('text/') || f.text
              ? 'lucide:file-text'
              : 'lucide:file'
          "
          class="text-default-500 size-4 shrink-0"
        />
        <span class="truncate">{{ f.filename || '附件' }}</span>
      </component>
    </div>
  </div>
</template>
