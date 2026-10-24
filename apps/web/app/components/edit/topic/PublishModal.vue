<script setup lang="ts">
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'
import { useTopicSubmitter } from '~/composables/topic/useTopicSubmitter'
import type { KunRadioOption } from '@kungal/ui-vue'

const open = defineModel<boolean>({ required: true })

const { section, isNSFW } = useTopicEditorStore()
const { rules, submit, isSubmitting, isRewriteMode } = useTopicSubmitter()
const { moemoepoint } = usePersistUserStore()

const needRules = computed(() => moemoepoint < 50)

const { groups, sectionOf } = useTopicSections()

const chosenKey = computed({
  get: () => section.value[0] ?? '',
  set: (value: string) => {
    section.value = [value]
  }
})
const chosen = computed(() => sectionOf(chosenKey.value))

const groupOptions = computed(() =>
  groups.value
    .map((group) => ({
      key: group.key,
      label: group.label,
      options: group.sections
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
    }))
    .filter((group) => group.options.length)
)

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
          选好分区就可以发布,
          封面图、投票与抽奖都是可选的。点击下方「确认发布」之前话题不会发出
        </p>
      </header>

      <div class="flex-1 space-y-8 overflow-y-auto px-1 py-1">
        <section class="space-y-3">
          <h3 class="flex items-center gap-2 font-semibold">
            <KunIcon name="lucide:layout-grid" class="h-5 w-5" />
            选择分区 <span class="text-danger-500">*</span>
          </h3>
          <div v-for="group in groupOptions" :key="group.key" class="space-y-2">
            <p class="text-default-500 text-sm">{{ group.label }}</p>
            <KunRadioGroup
              v-model="chosenKey"
              :options="group.options"
              :aria-label="group.label"
              variant="card"
              orientation="horizontal"
              size="sm"
              hide-indicator
            />
          </div>
          <p v-if="chosen" class="text-default-500 text-sm leading-relaxed">
            {{ chosen.description }}
          </p>
        </section>

        <EditTopicSpecialNotice />

        <section class="space-y-3">
          <h3 class="flex items-center gap-2 font-semibold">
            <KunIcon name="lucide:shield-alert" class="h-5 w-5" />
            NSFW 设置
          </h3>
          <KunCheckBox
            v-model="isNSFW"
            type="single"
            label="该话题包含 NSFW 内容 (R18 等)"
            color="primary"
          />
          <p class="text-default-500 text-sm">
            勾选后, 未开启网站 NSFW
            模式的用户将无法看到该话题。看起来不能在公司报告大会上放在 PPT
            里展示的话题都是 NSFW, 越严越好 (只允许萌萌的涩涩,
            不允许纯粹的色情废料)
          </p>
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
              v-model="rules.isAgreeCategory"
              type="single"
              label="我确认: 若话题为求助话题, 我已将求助性质的话题放入技术求助 / 寻求资源分区"
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

      <footer class="mt-4 flex items-center justify-end gap-3">
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton
          @click="submit"
          size="lg"
          :disabled="isSubmitting"
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
