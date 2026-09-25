<script setup lang="ts">
import type { GalgameCharacterTrait } from '~~/shared/types/galgame-character'

const props = defineProps<{
  traits: GalgameCharacterTrait[]
}>()

const isSpoilerRevealed = ref(false)
watch(
  () => props.traits,
  () => {
    isSpoilerRevealed.value = false
  }
)

const shown = computed(() =>
  isSpoilerRevealed.value
    ? props.traits
    : props.traits.filter((t) => t.spoiler === 0)
)
const hiddenCount = computed(
  () => props.traits.filter((t) => t.spoiler > 0).length
)

const groups = computed(() => {
  const out: { name: string; traits: GalgameCharacterTrait[] }[] = []
  for (const trait of shown.value) {
    const name = trait.group || '其他'
    const last = out.at(-1)
    if (last && last.name === name) {
      last.traits.push(trait)
    } else {
      out.push({ name, traits: [trait] })
    }
  }
  return out
})
</script>

<template>
  <div v-if="traits.length" class="space-y-2">
    <p class="text-default-500 flex items-center gap-1 text-xs">
      <KunIcon name="lucide:mouse-pointer-click" class="size-3.5 shrink-0" />
      点击属性, 查看拥有该属性的全部角色
    </p>

    <div v-for="group in groups" :key="group.name" class="space-y-1">
      <p class="text-default-400 text-xs">{{ group.name }}</p>
      <div class="flex flex-wrap gap-1.5">
        <KunLink
          v-for="trait in group.traits"
          :key="trait.id"
          underline="none"
          :to="`/galgame/trait/${trait.id}`"
        >
          <KunChip
            size="xs"
            :color="trait.spoiler > 0 ? 'warning' : 'default'"
            class-name="hover:bg-primary/15 hover:text-primary cursor-pointer transition-colors"
          >
            {{ trait.name }}<template v-if="trait.lie">（伪）</template>
          </KunChip>
        </KunLink>
      </div>
    </div>

    <KunButton
      v-if="hiddenCount && !isSpoilerRevealed"
      variant="flat"
      color="warning"
      size="sm"
      @click="isSpoilerRevealed = true"
    >
      <KunIcon name="lucide:eye" />
      显示 {{ hiddenCount }} 条剧透属性
    </KunButton>
  </div>
</template>
