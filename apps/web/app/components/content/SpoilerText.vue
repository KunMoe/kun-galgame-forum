<script setup lang="ts">
import type { ContentDocument } from '#shared/utils/api/schemas'

const props = withDefaults(
  defineProps<{
    content: string
    className?: string
  }>(),
  { className: '' }
)

const document = computed<ContentDocument>(() => ({
  object: 'document',
  children: [
    {
      object: 'paragraph',
      children: props.content.split(/\|\|(.*?)\|\|/gs).map((value, index) =>
        index % 2
          ? {
              object: 'inline_spoiler',
              children: [{ object: 'text', value }]
            }
          : { object: 'text', value }
      )
    }
  ]
}))
</script>

<template>
  <ContentDocument
    :key="content"
    :document="document"
    :class-name="cn('whitespace-pre-wrap wrap-anywhere', className)"
    compact
  />
</template>
