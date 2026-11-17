<script setup lang="ts">
const props = defineProps<{ groups: AgentSuggestionGroup[]; busy: boolean }>()
const emit = defineEmits<{ pick: [prompt: string, webSearch: boolean] }>()

const SHOWN = 3

const active = ref<number | null>(null)
const offset = ref(0)

const group = computed(() =>
  active.value === null ? null : props.groups[active.value]
)
const prompts = computed(() => {
  const all = group.value?.prompts ?? []
  if (all.length <= SHOWN) return all
  return Array.from(
    { length: SHOWN },
    (_, i) => all[(offset.value + i) % all.length]!
  )
})

const select = (i: number) => {
  active.value = active.value === i ? null : i
  offset.value = 0
}
const rotate = () => {
  const n = group.value?.prompts?.length ?? 0
  if (n) offset.value = (offset.value + SHOWN) % n
}
</script>

<template>
  <div class="flex flex-col items-center gap-3">
    <div class="flex flex-wrap justify-center gap-1.5">
      <KunButton
        v-for="(g, i) in groups"
        :key="g.label"
        size="sm"
        :variant="active === i ? 'flat' : 'light'"
        :color="active === i ? 'primary' : 'default'"
        :aria-pressed="active === i"
        @click="select(i)"
      >
        <KunIcon :name="`lucide:${g.icon}`" class="size-4" />
        {{ g.label }}
      </KunButton>
    </div>
    <div v-if="group" class="flex flex-wrap justify-center gap-2">
      <KunButton
        v-for="p in prompts"
        :key="p"
        size="sm"
        variant="bordered"
        :disabled="busy"
        @click="emit('pick', p, group.web_search)"
      >
        {{ p }}
      </KunButton>
      <KunButton
        v-if="(group.prompts?.length ?? 0) > SHOWN"
        size="sm"
        variant="light"
        @click="rotate"
      >
        <KunIcon name="lucide:refresh-cw" class="size-4" />
        换一组
      </KunButton>
    </div>
  </div>
</template>
