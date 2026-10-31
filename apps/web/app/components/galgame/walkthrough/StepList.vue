<script setup lang="ts">
import type {
  WalkthroughChart,
  WalkthroughStep
} from '#shared/utils/api/schemas'

const props = defineProps<{
  steps: WalkthroughStep[]
  chart: WalkthroughChart
  idPrefix?: string
  highlightedSave?: string | null
  size?: 'sm' | 'md'
}>()

const { point, short } = useWalkthroughLabels()

const groups = computed(() => groupWalkthroughSteps(props.steps))

const saveLabel = (key: string | null) => {
  const save = props.chart.save_points.find((s) => s.key === key)
  return save ? point(save.label, save.localized) : '存档'
}
</script>

<template>
  <div class="space-y-4">
    <section v-for="(g, i) in groups" :key="i" class="space-y-2">
      <div
        v-if="g.heading"
        class="text-default-600 flex items-center gap-2 font-medium"
      >
        <KunIcon name="lucide:bookmark" class-name="size-4 shrink-0" />
        <GalgameWalkthroughStepText
          :text="g.heading.step_text!"
          :localized="g.heading.localized"
          size="sm"
        />
      </div>
      <ol class="space-y-2">
        <li
          v-for="({ step, kind, no }, j) in g.items"
          :key="j"
          :id="
            idPrefix && step.save_key
              ? `${idPrefix}-save-${step.save_key}`
              : undefined
          "
          :class="
            cn(
              'flex scroll-mt-24 items-start gap-3 rounded-lg transition-colors duration-500',
              step.save_key &&
                highlightedSave === step.save_key &&
                'bg-primary-100'
            )
          "
        >
          <template v-if="kind === 'save'">
            <span
              class="bg-primary-500 mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full text-white"
            >
              <KunIcon name="lucide:save" class-name="size-3.5" />
            </span>
            <span class="text-primary-700 py-0.5 text-sm font-medium">
              在这里存档:「{{ saveLabel(step.save_key) }}」
            </span>
          </template>
          <template v-else>
            <span
              :class="
                cn(
                  'mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-medium',
                  kind === 'choice' && 'bg-default-100 text-default-700',
                  kind === 'action' && 'bg-warning-100 text-warning-700',
                  kind === 'note' && 'text-default-400'
                )
              "
            >
              <template v-if="kind === 'choice'">{{ no }}</template>
              <KunIcon
                v-else
                :name="kind === 'note' ? 'lucide:info' : 'lucide:gamepad-2'"
                class-name="size-3.5"
              />
            </span>
            <div
              :class="cn('min-w-0 space-y-1', kind === 'note' && 'opacity-80')"
            >
              <div class="flex flex-wrap items-center gap-2">
                <GalgameWalkthroughStepText
                  :text="step.step_text!"
                  :localized="step.localized"
                  :size="kind === 'note' ? 'sm' : size"
                />
                <KunChip
                  v-if="step.remark"
                  size="xs"
                  variant="flat"
                  color="primary"
                >
                  {{ short(step.remark, step.remark_localized) }}
                </KunChip>
                <KunChip
                  v-if="step.is_achievement_only"
                  size="xs"
                  color="warning"
                >
                  成就
                </KunChip>
              </div>
              <div
                v-if="step.alternatives.length"
                class="text-default-500 flex flex-wrap items-center gap-x-2 text-xs"
              >
                <span>差分 CG 回收时改选</span>
                <GalgameWalkthroughStepText
                  v-for="(alt, k) in step.alternatives"
                  :key="k"
                  :text="alt.text"
                  :localized="alt.localized"
                  size="sm"
                />
              </div>
            </div>
          </template>
        </li>
      </ol>
    </section>
  </div>
</template>
