<script setup lang="ts">
import type { KunRadioOption } from '@kungal/ui-vue'
import type { TopicClassification } from '#shared/utils/api/schemas'
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'

const props = defineProps<{
  classification: TopicClassification | null
  isClassifying: boolean
  isRewriteMode: boolean
}>()

const { section } = useTopicEditorStore()
const { sections, sectionOf } = useTopicSections()
const canMoveAny = useCan('topic.edit_any')

const chosenKey = computed({
  get: () => section.value[0] ?? '',
  set: (value: string) => {
    section.value = [value]
  }
})
const chosen = computed(() => sectionOf(chosenKey.value))

const mode = computed(() => {
  if (props.isRewriteMode) {
    return canMoveAny.value ? 'pick' : 'locked'
  }
  if (props.isClassifying) {
    return 'classifying'
  }
  return props.classification ? 'verdict' : 'pick'
})

const costText = (delta?: number) =>
  delta === undefined
    ? ''
    : delta < 0
      ? `发布消耗 ${-delta} 萌萌点`
      : `发布获得 ${delta} 萌萌点`

const options = computed(() =>
  sections.value
    .filter((s) => !s.is_archived || s.section === chosenKey.value)
    .map<KunRadioOption<string>>((s) => ({
      value: s.section,
      label: s.label,
      icon: s.icon_name,
      description:
        s.moemoepoint_on_create < 0
          ? `消耗 ${-s.moemoepoint_on_create} 萌萌点`
          : undefined
    }))
)
</script>

<template>
  <section class="space-y-3">
    <h3 class="flex items-center gap-2 font-semibold">
      <KunIcon name="lucide:layout-grid" class="h-5 w-5" />
      话题分区
      <span v-if="mode === 'pick'" class="text-danger-500">*</span>
    </h3>

    <div
      v-if="mode === 'classifying'"
      class="text-default-500 flex items-center gap-2 text-sm"
    >
      <KunIcon name="svg-spinners:90-ring-with-bg" />
      正在根据标题和正文判断话题分区...
    </div>

    <template v-else-if="mode === 'verdict' || mode === 'locked'">
      <div
        v-if="chosen"
        class="border-default-200 space-y-2 rounded-xl border p-4"
      >
        <div class="flex flex-wrap items-center gap-2">
          <KunChip :color="chosen.color">
            <KunIcon :name="chosen.icon_name" class="size-4 text-inherit" />
            {{ chosen.label }}
          </KunChip>
          <span
            :class="
              cn(
                'text-sm',
                chosen.moemoepoint_on_create < 0
                  ? 'text-warning-600'
                  : 'text-default-500'
              )
            "
          >
            {{ costText(chosen.moemoepoint_on_create) }}
          </span>
        </div>
        <p
          v-if="mode === 'verdict' && classification?.reason"
          class="text-default-600 text-sm"
        >
          判断依据: {{ classification.reason }}
        </p>
        <p class="text-default-500 text-sm leading-relaxed">
          {{ chosen.description }}
        </p>
      </div>
      <p class="text-default-500 text-xs">
        {{
          mode === 'verdict'
            ? '分区由系统根据话题内容判断。发布后如果认为分区不合适, 可以在话题页申请管理员调整'
            : '分区由系统决定, 如需调整, 请在话题页申请管理员调整'
        }}
      </p>
    </template>

    <template v-else>
      <p v-if="!isRewriteMode" class="text-default-500 text-sm">
        自动分区暂时不可用, 请手动选择话题所属的分区
      </p>
      <div class="space-y-2">
        <KunRadioGroup
          v-model="chosenKey"
          :options="options"
          aria-label="分区"
          variant="card"
          orientation="horizontal"
          size="sm"
          hide-indicator
        />
      </div>
      <p v-if="chosen" class="text-default-500 text-sm leading-relaxed">
        {{ chosen.description }}
      </p>
    </template>
  </section>
</template>
