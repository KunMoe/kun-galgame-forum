<script setup lang="ts">
import { useLocalStorage } from '@vueuse/core'
import { languageItems } from '~/constants/edit'

const { name } = storeToRefs(usePersistEditGalgameStore())

const introductionLanguage = ref<Language>('zh-cn')

const steps = [
  { n: 1, label: '基本信息' },
  { n: 2, label: '游戏介绍' },
  { n: 3, label: '分级元数据' },
  { n: 4, label: '预览图与提交' }
] as const

const totalSteps = steps.length

const step = useLocalStorage('kun-galgame-publish-step', 1)
onMounted(() => {
  if (step.value < 1 || step.value > totalSteps) {
    step.value = 1
  }
})

const goNext = () => {
  if (step.value < totalSteps) step.value++
}
const goPrev = () => {
  if (step.value > 1) step.value--
}
const goto = (n: number) => {
  step.value = n
}
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-6"
      >
        <KunHeader
          name="提交 Galgame 申请"
          description="提交后进入审核队列, 通过后才会公开。审核结果会通过站内消息通知您, 期间可在「我的提交」继续修改或撤回。"
        >
          <template #endContent>
            <div class="flex flex-col items-end gap-2">
              <KunLink to="/edit/galgame/mine">
                <KunButton size="sm" variant="flat">我的提交</KunButton>
              </KunLink>
              <KunLink
                target="_blank"
                to="/doc/galgame-publish-help"
                class="text-default-500 text-sm"
              >
                发布帮助文档
              </KunLink>
            </div>
          </template>
        </KunHeader>

        <KunInfo
          color="warning"
          title="此表单仅用于 VNDB 未收录的作品"
          description="鲲 Galgame 论坛已全量同步 VNDB。VNDB 收录的作品请回到「发布 Galgame」向导按名称搜索, 打开详情页发布资源, 不要在此重复提交。确实搜不到的原创 / 同人 / 独立作品再用本表单。"
        />

        <nav class="flex items-center">
          <template v-for="(s, i) in steps" :key="s.n">
            <button
              type="button"
              class="flex shrink-0 flex-col items-center gap-1 sm:flex-row sm:gap-2"
              @click="goto(s.n)"
            >
              <span
                :class="[
                  'flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors',
                  s.n === step
                    ? 'bg-primary text-white'
                    : s.n < step
                      ? 'bg-primary/15 text-primary'
                      : 'border-default-300 text-default-500 border'
                ]"
              >
                <KunIcon v-if="s.n < step" name="lucide:check" />
                <span v-else>{{ s.n }}</span>
              </span>
              <span
                :class="[
                  'hidden text-sm transition-colors sm:inline',
                  s.n === step
                    ? 'text-default-900 font-medium'
                    : 'text-default-500'
                ]"
              >
                {{ s.label }}
              </span>
            </button>
            <div
              v-if="i < steps.length - 1"
              :class="[
                'mx-2 h-px flex-1 transition-colors sm:mx-3',
                i < step - 1 ? 'bg-primary' : 'bg-default-200'
              ]"
            />
          </template>
        </nav>

        <KunDivider />

        <section v-show="step === 1" class="space-y-6">
          <div class="space-y-2">
            <h2 class="text-xl">游戏名</h2>
            <p class="text-default-500 text-sm">
              至少填写一种语言, 建议尽量补全 (没有官方译名的语言可留空)。
            </p>
            <div class="space-y-2">
              <KunInput placeholder="英语" v-model="name['en-us']" />
              <KunInput placeholder="日语" v-model="name['ja-jp']" />
              <KunInput placeholder="简体中文" v-model="name['zh-cn']" />
              <KunInput placeholder="繁体中文" v-model="name['zh-tw']" />
            </div>
          </div>

          <EditGalgamePrAlias type="create" />

          <EditGalgameStepHelp title="命名与别名建议">
            <p>
              游戏名以官方标题为准; 别名可填常用译名、罗马音、缩写等,
              方便其他用户搜索到这部作品。
            </p>
            <p>别名最多 17 个, 输入后按回车添加。</p>
          </EditGalgameStepHelp>
        </section>

        <section v-show="step === 2" class="space-y-4">
          <div class="space-y-1">
            <h2 class="text-xl">游戏介绍</h2>
            <p class="text-default-500 text-sm">至少填写一种语言的介绍。</p>
          </div>
          <KunTab
            :items="languageItems"
            v-model="introductionLanguage"
            variant="underlined"
            color="primary"
            size="sm"
          />
          <EditGalgameEditor :lang="introductionLanguage" type="publish" />

          <EditGalgameStepHelp title="介绍来源与图片规定">
            <p>
              介绍可参考官方页面、DLsite 的 ストーリー、Bangumi 或萌娘百科,
              也可以抛弃官方介绍自行撰写。
            </p>
            <p>
              请不要在介绍中放置任何 R18 图片。R18 认定很严格, 过于露骨的图片
              一律视为 R18。图片尽可能少放, 后续会有专门的图片展示位。
            </p>
          </EditGalgameStepHelp>
        </section>

        <section v-show="step === 3" class="space-y-6">
          <EditGalgameContentLimit />
          <EditGalgameMeta />
          <KunInfo
            color="info"
            title="标签 / 会社 / 引擎"
            description="标签、会社、引擎等元数据由 鲲 Galgame 论坛维护; 审核通过后可在 Galgame 详情页点击「编辑资料」补充。"
          />
        </section>

        <section v-show="step === 4" class="space-y-4">
          <EditGalgameBanner />
          <EditGalgameStepHelp title="预览图要求">
            <p>
              尽量在游戏内截取原始分辨率, 不要拍屏, 尽量不含设备窗口边框,
              宽度大于高度为佳。最终会被压缩到不大于 1920 × 1080。
            </p>
            <p>预览图同样不可包含 R18 等敏感内容。</p>
          </EditGalgameStepHelp>
        </section>

        <KunDivider />

        <div class="flex items-center justify-between gap-3">
          <KunButton variant="flat" :disabled="step === 1" @click="goPrev">
            上一步
          </KunButton>

          <KunButton v-if="step < totalSteps" @click="goNext">
            下一步
          </KunButton>
          <EditGalgameFooter v-else />
        </div>
      </KunCard>
    </ClientOnly>
  </div>
</template>
