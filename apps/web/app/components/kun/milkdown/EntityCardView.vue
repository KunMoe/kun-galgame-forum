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
  <ContentEntityCard
    v-if="typeof preview !== 'string'"
    :target="preview"
    inert
  />
  <div
    v-else-if="entity.type === 'tag' || entity.type === 'company'"
    class="border-default-200 h-9 w-48 rounded-full border"
    :class="preview === 'loading' && 'bg-default-100 animate-pulse'"
  />
  <div
    v-else
    :class="
      cn(
        'border-default-200 flex items-center gap-4 rounded-lg border p-3',
        'h-36'
      )
    "
  >
    <div
      :class="
        cn(
          'bg-default-100 h-30 w-[86px] shrink-0 rounded-lg',
          preview === 'loading' && 'animate-pulse'
        )
      "
    />
    <div class="flex min-w-0 flex-1 flex-col gap-2">
      <template v-if="preview === 'loading'">
        <div class="bg-default-100 h-3 w-20 animate-pulse rounded" />
        <div class="bg-default-100 h-4 w-3/4 animate-pulse rounded" />
        <div class="bg-default-100 h-3 w-1/2 animate-pulse rounded" />
      </template>
      <span v-else class="text-default-500 truncate text-sm">
        {{ entity.url }}
      </span>
    </div>
  </div>
</template>
