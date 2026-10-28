<script setup lang="ts">
const model = defineModel<string>({ required: true })

const tempReplyStore = useTempReplyStore()
const { pendingQuote } = storeToRefs(tempReplyStore)

const saveMarkdown = (editorMarkdown: string) => {
  model.value = editorMarkdown
}
</script>

<template>
  <div style="--kun-editor-min-height: 16rem">
    <KunMilkdownDualEditorProvider
      entities
      :value-markdown="model"
      :pending-quote="pendingQuote"
      @set-markdown="saveMarkdown"
      @quote-inserted="tempReplyStore.clearPendingQuote()"
    />
  </div>
</template>
