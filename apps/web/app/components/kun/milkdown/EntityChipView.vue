<script setup lang="ts">
import type { KunEntityViewProps } from '@kungal/editor-vue'
import { asEntityLinkRef } from '~/utils/entityRefType'

const props = defineProps<KunEntityViewProps>()

const linkRef = computed(() => asEntityLinkRef(props.entity))
const preview = linkRef.value
  ? useEntityPreview(() => linkRef.value!)
  : computed(() => 'failed' as const)
</script>

<template>
  <ContentEntityChip
    v-if="typeof preview !== 'string'"
    :target="preview"
    inert
  />
  <span
    v-else
    class="bg-default-100 text-default-500 inline-flex max-w-[min(100%,20em)] items-center rounded-md px-1.5 align-bottom text-[0.92em]"
  >
    <span class="truncate">{{ entity.url }}</span>
  </span>
</template>
