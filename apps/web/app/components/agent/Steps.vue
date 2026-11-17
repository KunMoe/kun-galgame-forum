<script setup lang="ts">
import type { AgentStep, AgentToolCall } from '~/utils/agentSteps'

const props = defineProps<{
  steps: AgentStep[]
  cards: AgentEntityCard[]
  running: boolean
}>()

const calls = computed(() => props.steps.flatMap((s) => s.calls))
const titles = computed(
  () =>
    new Map(
      [
        ...calls.value.flatMap((c) => c.output?.entities ?? []),
        ...props.cards
      ].map((c) => [agentCardKey(c), agentCardTitle(c)])
    )
)
const header = computed(() => {
  if (!props.running) {
    const failed = calls.value.filter((c) => c.error).length
    return `共运行 ${calls.value.length} 步${failed ? `，${failed} 步失败` : ''}`
  }
  const live = calls.value.find((c) => !c.done)
  return live ? `${describeToolCall(live, titles.value)}…` : '思考中…'
})

const open = ref<string | undefined>(props.running ? 'steps' : undefined)
watch(
  () => props.running,
  (r) => (open.value = r ? 'steps' : undefined)
)

const agent = useAgentStore()
const memory = useAgentMemory()
const linksMemory = (c: AgentToolCall) =>
  !!agent.persona?.memory_available &&
  c.done &&
  !!agentMemoryChange(c.output?.meta)

const expanded = ref(new Set<string>())
const hasFinds = (c: AgentToolCall) =>
  !!c.output?.entities.length ||
  !!c.output?.links.length ||
  !!agentPastConversations(c.output?.meta).length
const toggle = (id: string) => {
  const next = new Set(expanded.value)
  if (!next.delete(id)) next.add(id)
  expanded.value = next
}
</script>

<template>
  <KunAccordion v-model="open" variant="light">
    <KunAccordionItem value="steps">
      <template #title>
        <span class="text-default-500 flex items-center gap-1.5 text-sm">
          <KunIcon
            :name="running ? 'lucide:loader-circle' : 'lucide:list-checks'"
            :class="cn('size-4 shrink-0', running && 'animate-spin')"
          />
          {{ header }}
        </span>
      </template>
      <ol class="border-default-200 ml-2 flex flex-col gap-2 border-l pl-3">
        <li v-for="s in steps" :key="s.key" class="flex flex-col gap-1">
          <p
            v-for="n in s.notes"
            :key="n.id"
            :class="
              cn(
                'text-default-500 text-sm leading-relaxed whitespace-pre-wrap',
                n.type === 'reasoning' && 'line-clamp-4'
              )
            "
          >
            {{ n.text.trim() }}
          </p>
          <div v-for="c in s.calls" :key="c.id">
            <component
              :is="hasFinds(c) ? 'button' : 'p'"
              :type="hasFinds(c) ? 'button' : undefined"
              :aria-expanded="hasFinds(c) ? expanded.has(c.id) : undefined"
              :class="
                cn(
                  'text-default-600 flex w-full items-start gap-1.5 text-left text-sm',
                  hasFinds(c) && 'hover:text-foreground cursor-pointer'
                )
              "
              @click="hasFinds(c) && toggle(c.id)"
            >
              <KunIcon
                :name="
                  c.error
                    ? 'lucide:circle-x'
                    : c.done
                      ? 'lucide:check'
                      : 'lucide:loader-circle'
                "
                :class="
                  cn(
                    'mt-0.5 size-4 shrink-0',
                    c.error
                      ? 'text-danger'
                      : c.done
                        ? 'text-success'
                        : 'animate-spin'
                  )
                "
              />
              <span :class="cn('min-w-0 flex-1', c.error && 'text-danger/80')">
                {{ describeToolCall(c, titles) }}
                <button
                  v-if="linksMemory(c) && !hasFinds(c)"
                  type="button"
                  class="text-primary ml-1 text-xs hover:underline"
                  @click="memory.open()"
                >
                  查看记忆
                </button>
              </span>
              <KunIcon
                v-if="hasFinds(c)"
                name="lucide:chevron-down"
                :class="
                  cn(
                    'text-default-400 mt-0.5 size-4 shrink-0 transition-transform',
                    expanded.has(c.id) && 'rotate-180'
                  )
                "
              />
            </component>
            <AgentStepFinds
              v-if="c.output && expanded.has(c.id)"
              :output="c.output"
            />
          </div>
        </li>
      </ol>
    </KunAccordionItem>
  </KunAccordion>
</template>
