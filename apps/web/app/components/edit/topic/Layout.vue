<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import type { KunEditorExpose, KunHeading } from '@kungal/editor-vue'
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'
import { KUN_EDITOR_TOOLBAR_ITEMS } from '~/constants/editor'
const tempStore = useTempEditStore()
const { content, polls, lotteries } = useTopicEditorStore()
const adapters = useKunEditorAdapters()

const headings = ref<KunHeading[]>([])
const editorRef = ref<KunEditorExpose | null>(null)

const isPublishOpen = ref(false)
const openPublish = () => {
  isPublishOpen.value = true
}

const isDraftOpen = ref(false)

const attachedText = computed(() =>
  [
    polls.value.length && `${polls.value.length} 个投票`,
    lotteries.value.length && `${lotteries.value.length} 个抽奖`
  ]
    .filter(Boolean)
    .join('、')
)

type EditorViewMode = 'wysiwyg' | 'source' | 'split'
const editorMode = ref<EditorViewMode>('wysiwyg')
const onSetMode = (
  mode: EditorViewMode,
  apply: (mode: EditorViewMode) => void
) => {
  editorMode.value = mode
  apply(mode)
}

const isMobile = useMediaQuery('(max-width: 767px)')
const editorViews = computed<EditorViewMode[]>(() =>
  isMobile.value ? ['wysiwyg', 'source'] : ['wysiwyg', 'source', 'split']
)
watch(isMobile, (mobile) => {
  if (mobile && editorMode.value === 'split') {
    editorMode.value = 'wysiwyg'
  }
})

const onKeydown = (event: KeyboardEvent) => {
  if (event.ctrlKey && event.key === 'Enter') {
    event.preventDefault()
    openPublish()
  }
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))

onBeforeRouteLeave(async () => {
  if (tempStore.isTopicRewriting) {
    const res =
      await useComponentMessageStore().alert(
        '确认离开界面吗？您的更改将不会保存。'
      )
    if (!res) {
      return false
    }
    tempStore.resetRewriteTopicData()
  }
})
</script>

<template>
  <div>
    <div
      class="et-topbar border-default-200 bg-background/90 sticky top-[72px] z-20 mb-4 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border px-3 py-2 backdrop-blur"
    >
      <div id="et-viewswitch" class="shrink-0" />
      <span class="bg-default-200 hidden h-5 w-px shrink-0 sm:block" />
      <KunScrollShadow
        axis="horizontal"
        shadow-size="1.5rem"
        shadow-color="var(--color-default-100)"
        draggable
        scroll-buttons
        class="bg-default-100 order-last w-full rounded-lg px-1 py-1 sm:order-none sm:w-auto sm:min-w-0 sm:flex-1 sm:rounded-none sm:bg-transparent sm:p-0"
      >
        <div id="et-toolbar" class="flex w-max items-center" />
      </KunScrollShadow>
      <div
        class="order-1 ml-auto flex shrink-0 items-center gap-2 sm:order-none sm:ml-0"
      >
        <KunButton
          v-if="!tempStore.isTopicRewriting"
          variant="light"
          color="primary"
          @click="isDraftOpen = true"
        >
          <KunIcon name="lucide:notebook-pen" class="mr-1 h-4 w-4" />
          草稿
        </KunButton>
        <KunButton color="primary" @click="openPublish">
          <KunIcon name="lucide:send" class="mr-1 h-4 w-4" />
          发布话题
        </KunButton>
      </div>
    </div>

    <p
      v-if="!tempStore.isTopicRewriting"
      class="text-default-500 mb-4 flex items-start gap-1.5 px-1 text-sm"
    >
      <KunIcon name="lucide:info" class="mt-0.5 h-4 w-4 shrink-0" />
      <span>
        写完正文后点击「发布话题」, 会先弹出设置窗口: 在那里选择分区,
        设置封面图, 添加投票与抽奖。确认之前话题不会发出
        <span v-if="attachedText" class="text-primary-500">
          · 已添加 {{ attachedText }}, 发布时一并创建
        </span>
      </span>
    </p>

    <div class="flex justify-center gap-6">
      <aside v-if="headings.length" class="hidden w-48 shrink-0 lg:block">
        <nav class="sticky top-32 space-y-1 py-1 text-sm">
          <button
            v-for="(h, i) in headings"
            :key="i"
            type="button"
            :style="{ paddingInlineStart: `${(h.level - 1) * 0.75}rem` }"
            :title="h.text"
            class="text-default-400 hover:text-primary-500 block w-full truncate text-left transition-colors"
            @click="editorRef?.scrollToHeading(i)"
          >
            {{ h.text || '(无标题)' }}
          </button>
        </nav>
      </aside>

      <div
        :class="
          cn('min-w-0', editorMode === 'split' ? 'flex-1' : 'w-full max-w-3xl')
        "
      >
        <div class="bg-content1 shadow-kun-md px-6 sm:px-16">
          <div
            class="border-default-200/60 border-b pt-8 pb-4 sm:pt-14 sm:pb-5"
          >
            <EditTopicTitle />
          </div>

          <div class="pt-5 pb-10 sm:pt-7 sm:pb-20">
            <ClientOnly>
              <KunEditor
                ref="editorRef"
                v-model="content"
                :adapters="adapters"
                :views="editorViews"
                placeholder="在此输入您的话题正文..."
                class="kun-doc-editor"
                @update:headings="headings = $event"
              >
                <template #view-switch="api">
                  <Teleport to="#et-viewswitch" defer>
                    <KunEditorViewSwitch
                      v-bind="api"
                      :set-mode="(mode) => onSetMode(mode, api.setMode)"
                    />
                  </Teleport>
                </template>
                <template #toolbar="api">
                  <Teleport to="#et-toolbar" defer>
                    <template v-if="editorMode === 'wysiwyg'">
                      <KunEditorToolbar
                        v-bind="api"
                        :items="KUN_EDITOR_TOOLBAR_ITEMS"
                      />
                      <span
                        class="bg-default-200 mx-1 h-5 w-px"
                        aria-hidden="true"
                      />
                      <KunMilkdownImageDialog v-bind="api" />
                    </template>
                  </Teleport>
                </template>
              </KunEditor>
            </ClientOnly>
          </div>
        </div>
      </div>
    </div>

    <EditTopicPublishModal v-model="isPublishOpen" />
    <EditTopicDraftModal v-model="isDraftOpen" />
  </div>
</template>

<!-- Global (not scoped): the toolbar is TELEPORTED into #et-toolbar, so a scoped
     rule wouldn't reach it. <KunEditorToolbar>'s root is `flex flex-wrap`; force
     a single row here so #et-toolbar's overflow-x-auto scrolls it instead of it
     stacking into multiple tall rows (the mobile problem). -->
<style>
#et-toolbar > * {
  flex-wrap: nowrap;
}
</style>
