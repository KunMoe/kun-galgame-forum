<script setup lang="ts">
import type { GalgameTag } from '#shared/utils/api/schemas'
import { ancestorTrail } from '~/utils/galgame/tagTree'

const props = defineProps<{ tag: GalgameTag; name: string }>()

const { allowsNsfw } = useContentStance()

const trail = computed(() =>
  ancestorTrail(props.tag.parent_tags, props.tag.ancestors).map((node) => ({
    id: node.id,
    name: catalogVocabularyName(node),
    linked: allowsNsfw.value || !node.is_sexual
  }))
)
const otherParents = computed(() => props.tag.parent_tags.slice(1))
</script>

<template>
  <div
    v-if="trail.length"
    class="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-sm"
  >
    <span class="text-default-500 mr-1">层级</span>
    <template v-for="node in trail" :key="node.id">
      <KunLink
        v-if="node.linked"
        underline="none"
        color="default"
        :to="`/galgame/tag/${node.id}`"
        class="hover:text-primary"
      >
        {{ node.name }}
      </KunLink>
      <span v-else class="text-default-500">{{ node.name }}</span>
      <KunIcon name="lucide:chevron-right" class="text-default-300 size-3.5" />
    </template>
    <span class="text-foreground font-medium">{{ name }}</span>

    <template v-if="otherParents.length">
      <span class="text-default-500 mr-1 ml-3">也属于</span>
      <KunLink
        v-for="parent in otherParents"
        :key="parent.id"
        underline="none"
        :to="`/galgame/tag/${parent.id}`"
      >
        <KunChip
          size="xs"
          variant="flat"
          :color="parent.is_sexual ? 'danger' : 'default'"
          class-name="cursor-pointer"
        >
          {{ catalogVocabularyName(parent) }}
        </KunChip>
      </KunLink>
    </template>
  </div>
</template>
