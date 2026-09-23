<script setup lang="ts">
import type { WorkDigest, WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{ work: WorkRef; digest: WorkDigest | null }>()

const nameOf = useWorkName()

const madeBy = computed(() => {
  const parts: string[] = []
  if (props.digest?.developer_names.length) {
    parts.push(`由 ${props.digest.developer_names.join('、')} 制作`)
  }
  if (props.digest?.release) {
    parts.push(`发售于 ${props.digest.release}`)
  }
  return parts.join('，')
})
</script>

<template>
  <div class="flex items-start gap-3">
    <ActivityCardWorkCover :work="work" />

    <div class="min-w-0 flex-1 space-y-1">
      <KunLink
        underline="none"
        color="default"
        :to="`/galgame/${work.id}`"
        class-name="hover:text-primary block"
      >
        <h3 class="line-clamp-2 font-medium break-all">
          {{ nameOf(work) }}
        </h3>
      </KunLink>
      <p v-if="madeBy" class="text-default-700 text-sm break-all">
        {{ madeBy }}
      </p>
      <p
        v-if="digest?.intro_excerpt"
        class="text-default-500 line-clamp-3 text-sm break-all"
      >
        {{ markdownToText(digest.intro_excerpt) }}
      </p>
    </div>
  </div>
</template>
