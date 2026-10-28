<script setup lang="ts">
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'
import { useTopicSubmitter } from '~/composables/topic/useTopicSubmitter'

const open = defineModel<boolean>({ required: true })

const { isNSFW } = useTopicEditorStore()
const {
  rules,
  submit,
  isSubmitting,
  isRewriteMode,
  classification,
  isClassifying,
  classifyIssue,
  classify
} = useTopicSubmitter()
const { moemoepoint } = usePersistUserStore()

const needRules = computed(() => moemoepoint < 50)
const canMoveAny = useCan('topic.edit_any')
const { sectionOf } = useTopicSections()

const showSectionField = computed(() =>
  isRewriteMode.value ? canMoveAny.value : !!classifyIssue.value
)
const decided = computed(() =>
  classification.value ? sectionOf(classification.value.section) : undefined
)
const isJudgedNSFW = computed(() => !!classification.value?.is_nsfw)
watch(isJudgedNSFW, (judged) => {
  if (judged) isNSFW.value = true
})

watch(open, (isOpen) => {
  if (isOpen && !isRewriteMode.value) {
    classify()
  }
})

const confirmText = computed(() => {
  if (isSubmitting.value) return '处理中...'
  return isRewriteMode.value ? '确认更改' : '确认发布'
})
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-2xl">
    <div class="flex max-h-[80vh] flex-col">
      <header class="mb-4 space-y-1">
        <div class="flex items-center gap-2">
          <KunIcon name="lucide:send" class="text-primary-500 h-5 w-5" />
          <h2 class="text-xl font-bold">
            {{ isRewriteMode ? '提交更改' : '发布话题' }}
          </h2>
        </div>
        <p v-if="!isRewriteMode" class="text-default-500 text-sm">
          分区和是否 NSFW 会根据话题内容自动判断,
          封面图、投票与抽奖都是可选的。点击下方「确认发布」之前话题不会发出
        </p>
      </header>

      <div class="flex-1 space-y-8 overflow-y-auto px-1 py-1">
        <EditTopicSectionField
          v-if="showSectionField"
          :classification="classification"
          :is-classifying="isClassifying"
          :classify-issue="classifyIssue"
          @reclassify="classify"
          :is-rewrite-mode="isRewriteMode"
        />

        <EditTopicSpecialNotice />

        <section class="space-y-3">
          <h3 class="flex items-center gap-2 font-semibold">
            <KunIcon name="lucide:shield-alert" class="h-5 w-5" />
            NSFW
          </h3>
          <p
            v-if="isJudgedNSFW"
            class="bg-danger-50 text-danger-600 rounded-lg px-3 py-2 text-sm"
          >
            系统根据内容判定这是 NSFW 话题, 发布后未开启网站 NSFW
            模式的用户将看不到它
          </p>
          <template v-else>
            <KunCheckBox
              v-model="isNSFW"
              type="single"
              label="该话题包含 NSFW 内容 (R18 等)"
              color="primary"
            />
            <p class="text-default-500 text-sm">
              系统会根据标题、正文和图片自动判断, 判定为 NSFW
              的话题无法取消标记。你也可以主动标记,
              看起来不能在公司报告大会上放在 PPT 里展示的话题都是 NSFW
            </p>
          </template>
        </section>

        <EditTopicAccessScope />

        <section class="space-y-3">
          <EditTopicCoverPicker />
        </section>

        <EditTopicMiniApps v-if="!isRewriteMode" />

        <section v-if="needRules" class="space-y-3">
          <h3 class="flex items-center gap-2 font-semibold">
            <KunIcon name="lucide:circle-alert" class="h-5 w-5" />
            发布须知 <span class="text-danger-500">*</span>
          </h3>
          <p class="text-default-500 text-sm">
            勾选确认下面的须知即可发布话题, 仅对萌萌点 &lt; 50 的用户生效,
            这是为了防止发布过多无意义的话题, 请您理解
          </p>
          <div class="space-y-2">
            <KunCheckBox
              v-model="rules.isReadRule"
              type="single"
              label="我确认: 我已完整阅读过话题发布规定"
              color="primary"
            />
            <KunCheckBox
              v-model="rules.isValidTitle"
              type="single"
              label="我确认: 我话题的标题非常直观具体详细, 而绝对不是《游戏报错求助》，《求助资源》这类模糊标题"
              color="primary"
            />
            <KunCheckBox
              v-model="rules.isKnownConsequence"
              type="single"
              label="我确认: 违反上面的规则或违反论坛的总规定, 将会被扣除巨量萌萌点甚至永久封禁"
              color="primary"
            />
          </div>
        </section>
      </div>

      <footer class="mt-4 flex flex-wrap items-center justify-end gap-3">
        <p
          v-if="!isRewriteMode"
          class="text-default-500 mr-auto flex items-center gap-1.5 text-sm"
        >
          <template v-if="isClassifying">
            <KunIcon name="svg-spinners:90-ring-with-bg" />
            正在判断分区…
          </template>
          <template v-else-if="decided">
            发布到「{{ decided.label }}」
            <span
              v-if="decided.moemoepoint_on_create < 0"
              class="text-warning-600"
            >
              , 消耗 {{ -decided.moemoepoint_on_create }} 萌萌点
            </span>
          </template>
        </p>
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton
          @click="submit"
          size="lg"
          :disabled="isSubmitting || isClassifying"
          class="min-w-[120px]"
        >
          <KunIcon
            v-if="isSubmitting"
            name="lucide:loader-2"
            class="h-5 w-5 animate-spin"
          />
          <span v-else>{{ confirmText }}</span>
        </KunButton>
      </footer>
    </div>
  </KunModal>
</template>
