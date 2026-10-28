<script setup lang="ts">
import type { KunEditorExpose } from '@kungal/editor-vue'
import {
  KUN_EDITOR_ENTITY_VIEW,
  KUN_EDITOR_TOOLBAR_ITEMS,
  KUN_EDITOR_TOOLBAR_OVERRIDES
} from '~/constants/editor'
import type { ReplyReference } from '~/store/types/topic/reply'

const props = defineProps<{
  valueMarkdown: string
  language?: Language
  disableImage?: boolean
  entities?: boolean
  pendingQuote?: ReplyReference | null
  placeholder?: string
}>()

const emits = defineEmits<{
  setMarkdown: [value: string]
  quoteInserted: []
}>()

const model = computed({
  get: () => props.valueMarkdown,
  set: (value) => emits('setMarkdown', value)
})

const adapters = useKunEditorAdapters({
  image: props.disableImage !== true,
  entities: props.entities
})

const features = { quote: true, sticker: props.disableImage !== true }

const editorRef = ref<KunEditorExpose | null>(null)

const insertQuote = (q: ReplyReference) => {
  editorRef.value?.insertMention({ userId: q.userId, name: q.userName })
  editorRef.value?.insertQuote({
    refId: String(q.replyId),
    label: `#${q.floor}`
  })
  emits('quoteInserted')
}

const consumePendingQuote = () => {
  const q = props.pendingQuote
  if (q) {
    requestAnimationFrame(() => insertQuote(q))
  }
}

watch(() => props.pendingQuote, consumePendingQuote)
onMounted(consumePendingQuote)

const textCount = computed(() => markdownToText(props.valueMarkdown).length)
</script>

<template>
  <div class="space-y-3">
    <KunEditor
      ref="editorRef"
      v-model="model"
      :adapters="adapters"
      :features="features"
      :entity-view="KUN_EDITOR_ENTITY_VIEW"
      :locale="language ?? 'zh-cn'"
      :placeholder="placeholder"
      :views="['wysiwyg', 'source']"
    >
      <template #view-switch="api">
        <KunEditorViewSwitch v-bind="api" />
      </template>
      <template #toolbar="api">
        <div class="flex flex-wrap items-center gap-1">
          <KunEditorToolbar
            v-bind="api"
            :items="KUN_EDITOR_TOOLBAR_ITEMS"
            :item-overrides="KUN_EDITOR_TOOLBAR_OVERRIDES"
          />
          <template v-if="disableImage !== true">
            <span class="bg-default-200 mx-1 h-5 w-px" aria-hidden="true" />
            <KunMilkdownImageDialog v-bind="api" />
          </template>
        </div>
      </template>
    </KunEditor>

    <div class="flex items-center justify-between text-sm">
      <slot />

      <div class="flex shrink-0 items-center gap-2">
        <KunChip color="success">
          <KunIcon
            name="simple-icons:markdown"
            class="text-success-700 dark:text-success"
          />
          Markdown 支持
        </KunChip>
        <span>
          {{ `${textCount} 字` }}
        </span>
      </div>
    </div>

    <div class="text-default-500 text-sm">
      特殊语法: 您可以使用 ||隐藏文本|| 来隐藏图片或者文字 (目前依然禁止 R18
      图片内容)
    </div>
  </div>
</template>
